package aiservice

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/dbdomain"
	aireq "bluebell/internal/dto/request/ai"
	ai_resp "bluebell/internal/dto/response/ai"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type AIServiceStruct struct {
	db    dbdomain.RemarkRepository
	model *openai.ChatModel
}

func NewaiSvc(db dbdomain.RemarkRepository) (*AIServiceStruct, error) {
	// 从配置获取魔搭社区配置
	msConfig := config.Get().ModelScope
	if msConfig == nil || msConfig.Api_key == "" || msConfig.Url == "" || msConfig.Modle == "" {
		return nil, fmt.Errorf("Modelscope configuration is incomplete")
	}

	// 创建 ChatModel
	model, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:   msConfig.Modle,
		APIKey:  msConfig.Api_key,
		BaseURL: msConfig.Url,
	})
	if err != nil {
		return nil, err
	}

	return &AIServiceStruct{
		db:    db,
		model: model,
	}, nil
}

func (s *AIServiceStruct) RemarkSummary(ctx context.Context, req *aireq.RemarkSummaryReq) (ai_resp.RemarkSummaryResp, error) {
	// 1. 获取评论
	remarks, err := s.db.GetRemarksByPostID(ctx, req.PostID)
	if err != nil {
		return ai_resp.RemarkSummaryResp{}, err
	}

	if len(remarks) == 0 {
		return ai_resp.RemarkSummaryResp{
			Summary: "暂无评论",
		}, nil
	}

	// 限制评论数量，防止 token 超限
	if req.MaxComments > 0 && len(remarks) > req.MaxComments {
		remarks = remarks[:req.MaxComments]
	}

	// 2. 组装评论内容
	var comments strings.Builder
	for _, r := range remarks {
		comments.WriteString("- " + r.Content + "\n")
	}

	// 3. 构建 Prompt
	prompt := "请用50-100字简洁总结以下评论的主要内容，要体现整体情感倾向（正面/中性/负面）：\n\n" + comments.String()

	// 4. 调用 LLM 生成总结
	resp, err := s.model.Generate(ctx, []*schema.Message{
		schema.UserMessage(prompt),
	})

	if err != nil {
		return ai_resp.RemarkSummaryResp{}, err
	}

	// 获取当前时间
	now := time.Now().Format("2006-01-02 15:04:05")

	return ai_resp.RemarkSummaryResp{
		Summary:      resp.Content,
		SummaryType:  "brief",
		Language:     "zh",
		PostID:       req.PostID,
		CommentCount: len(remarks),
		CreatedAt:    now,
	}, nil
}
