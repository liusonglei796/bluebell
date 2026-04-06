package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bluebell/internal/config"
	"bluebell/pkg/errorx"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// AuditOutput 审核输出
// Called by: mq/audit_consumer.go (HandleDelivery 中读取 result.IsSafe/Violations/Score/Reason)
type AuditOutput struct {
	IsSafe     bool     `json:"is_safe"`
	Violations []string `json:"violations,omitempty"`
	Score      int      `json:"score"` // 0-100, 100最安全
	Reason     string   `json:"reason"`
}

// Auditor AI 内容审核服务
type Auditor struct {
	runnable compose.Runnable[map[string]any, *AuditOutput]
	enabled  bool
}

// NewAuditor 创建审核服务实例
// Called by: cmd/bluebell/main.go (line 129: ai.NewAuditor(ctx, cfg))
func NewAuditor(ctx context.Context, cfg *config.Config) (*Auditor, error) {
	aiCfg := cfg.AIAudit
	if aiCfg == nil || !aiCfg.Enabled {
		zap.L().Info("AI audit is disabled by config")
		return &Auditor{enabled: false}, nil
	}

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: aiCfg.BaseURL,
		APIKey:  aiCfg.APIKey,
		Model:   aiCfg.Model,
	})
	if err != nil {
		return nil, err
	}

	zap.L().Info("AI audit client initialized",
		zap.String("model", aiCfg.Model),
		zap.String("base_url", aiCfg.BaseURL))

	runnable, err := buildAuditWorkflow(ctx, cm)
	if err != nil {
		return nil, err
	}

	zap.L().Info("AI auditor initialized")

	return &Auditor{
		runnable: runnable,
		enabled:  true,
	}, nil
}

// IsEnabled 返回审核服务是否已启用
// Called by: mq/audit_consumer.go (HandleDelivery 中 c.auditor.IsEnabled() 判断是否跳过审核)
func (a *Auditor) IsEnabled() bool {
	return a.enabled
}

// Audit 执行内容审核
// Called by: mq/audit_consumer.go (HandleDelivery 中 c.auditor.Audit(ctx, msg.Title, msg.Content))
func (a *Auditor) Audit(ctx context.Context, title, content string) (*AuditOutput, error) {
	if !a.enabled {
		return &AuditOutput{IsSafe: true, Score: 100}, nil
	}

	input := map[string]any{
		"title":   title,
		"content": content,
	}
	if strings.TrimSpace(title) == "" {
		input["title"] = "无"
	}

	result, err := a.runnable.Invoke(ctx, input)
	if err != nil {
		zap.L().Error("AI audit failed", zap.Error(err))
		return nil, err
	}

	zap.L().Info("AI audit result",
		zap.Bool("is_safe", result.IsSafe),
		zap.Int("score", result.Score),
		zap.Strings("violations", result.Violations))

	return result, nil
}

// buildAuditWorkflow 构建审核工作流 Graph（包内私有）
// Called by: NewAuditor (line 56: buildAuditWorkflow(ctx, cm))
func buildAuditWorkflow(ctx context.Context, chatModel model.ToolCallingChatModel) (compose.Runnable[map[string]any, *AuditOutput], error) {
	const (
		nodeOfPrompt = "prompt"
		nodeOfModel  = "model"
		nodeOfParse  = "parse"
	)

	systemPrompt := `你是一个专业的内容审核助手。请审核以下内容是否包含违规信息。

审核规则：
1. 政治敏感内容
2. 色情内容
3. 暴力恐怖内容
4. 辱骂和仇恨言论
5. 垃圾广告和诈骗信息

请以 JSON 格式返回审核结果，格式如下：
{
    "is_safe": true/false,
    "violations": ["违规类型1", "违规类型2"],
    "score": 0-100,
    "reason": "审核原因"
}

只返回 JSON，不要包含其他文字。`

	chatTpl := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(systemPrompt),
		schema.UserMessage("标题: {title}\n内容: {content}\n请审核并返回JSON格式的审核结果。"),
	)

	parseLambda := compose.InvokableLambda(func(ctx context.Context, airespInput *schema.Message) (*AuditOutput, error) {
		var result AuditOutput
		content := airespInput.Content
		if strings.Contains(content, "```") {
			start := strings.Index(content, "{")
			end := strings.LastIndex(content, "}")
			if start >= 0 && end > start {
				content = content[start : end+1]
			}
		}
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "parse audit response failed, content: %s", content)
		}
		return &result, nil
	})

	g := compose.NewGraph[map[string]any, *AuditOutput]()

	_ = g.AddChatTemplateNode(nodeOfPrompt, chatTpl)
	_ = g.AddChatModelNode(nodeOfModel, chatModel)
	_ = g.AddLambdaNode(nodeOfParse, parseLambda)

	_ = g.AddEdge(compose.START, nodeOfPrompt)
	_ = g.AddEdge(nodeOfPrompt, nodeOfModel)
	_ = g.AddEdge(nodeOfModel, nodeOfParse)
	_ = g.AddEdge(nodeOfParse, compose.END)

	runnable, err := g.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile audit workflow failed: %w", err)
	}

	return runnable, nil
}
