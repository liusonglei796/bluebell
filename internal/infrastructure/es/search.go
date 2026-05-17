package es

import (
	"bytes"         // 提供字节缓冲区操作，用于构建 HTTP 请求体
	"context"       // 提供上下文支持，用于请求取消和超时控制
	"encoding/json" // 提供 JSON 序列化和反序列化
	"fmt"
	"io"            // 提供 I/O 操作接口

	"github.com/elastic/go-elasticsearch/v8/esapi" // ES 官方 Go 客户端 API 类型定义
	"go.uber.org/zap"                              // 高性能日志库
)

// SearchRequest 搜索请求结构体
// 用于接收前端传来的搜索参数，包含关键词和分页信息
type SearchRequest struct {
	Keyword  string `json:"keyword" binding:"required"` // 搜索关键词（必填字段）
	Page     int    `json:"page"`                       // 当前页码（从 1 开始，默认 1）
	PageSize int    `json:"page_size"`                  // 每页条数（默认 20，最大 50）
}

// SearchResponse 搜索响应结构体
// 返回给前端的标准化搜索结果
type SearchResponse struct {
	Total    int64           `json:"total"`     // 符合条件的总文档数
	Page     int             `json:"page"`      // 当前页码
	PageSize int             `json:"page_size"` // 每页条数
	Posts    []SearchPostDoc `json:"posts"`     // 搜索结果文档列表
}

// SearchPostDoc 搜索结果文档结构体
// 表示 ES 中返回的单条帖子文档，包含原始字段和高亮字段
type SearchPostDoc struct {
	PostID           string   `json:"post_id"`                     // 帖子唯一 ID
	AuthorID         int64    `json:"author_id"`                   // 作者用户 ID
	CommunityID      int64    `json:"community_id"`                // 所属社区 ID
	PostTitle        string   `json:"post_title"`                  // 帖子标题（原始文本）
	Content          string   `json:"content"`                     // 帖子内容（原始文本）
	Status           int8     `json:"status"`                      // 帖子状态（1=已发布，0=已删除）
	CreatedAt        string   `json:"created_at"`                  // 创建时间（RFC3339 格式）
	HighlightTitle   []string `json:"highlight_title,omitempty"`   // 标题高亮片段（omitempty 表示为空时不返回）
	HighlightContent []string `json:"highlight_content,omitempty"` // 内容高亮片段（omitempty 表示为空时不返回）
}

// Search 执行全文搜索，支持高亮和分页
// 这是 ES 搜索功能的核心入口方法
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	// ====== 第一步：参数校验和默认值处理 ======
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 50 {
		req.PageSize = 20
	}

	// ====== 第二步：构建 ES 查询 DSL ======
	query := buildSearchQuery(req)

	// ====== 第三步：序列化查询 DSL 为 JSON ======
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal search query failed: %w", err)
	}

	// ====== 第四步：记录搜索日志 ======
	zap.L().Info("ES search",
		zap.String("keyword", req.Keyword),
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
	)

	// ====== 第五步：执行 ES 搜索请求 ======
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexPost),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("ES search request failed: %w", err)
	}
	defer res.Body.Close()

	// ====== 第六步：检查 ES 响应状态 ======
	if res.IsError() {
		respBody, _ := io.ReadAll(res.Body)
		zap.L().Error("ES search error response",
			zap.Int("status", res.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("ES search error: %s", string(respBody))
	}

	// ====== 第七步：解析 ES 响应 ======
	resp, err := parseSearchResponse(res)
	if err != nil {
		return nil, err
	}

	// ====== 第八步：填充分页信息 ======
	resp.Page = req.Page
	resp.PageSize = req.PageSize

	return resp, nil
}

// buildSearchQuery 构建 ES 搜索 DSL
func buildSearchQuery(req *SearchRequest) map[string]interface{} {
	from := (req.Page - 1) * req.PageSize

	query := map[string]interface{}{
		"from": from,
		"size": req.PageSize,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": req.Keyword,
				"fields": []string{
					"post_title^2",
					"content",
				},
				"type": "best_fields",
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"post_title": map[string]interface{}{
					"pre_tags":  []string{"<em class='highlight'>"},
					"post_tags": []string{"</em>"},
				},
				"content": map[string]interface{}{
					"pre_tags":            []string{"<em class='highlight'>"},
					"post_tags":           []string{"</em>"},
					"fragment_size":       150,
					"number_of_fragments": 3,
				},
			},
		},
		"_source": map[string]interface{}{
			"includes": []string{
				"post_id",
				"author_id",
				"community_id",
				"post_title",
				"content",
				"status",
				"created_at",
			},
		},
		"sort": []map[string]interface{}{
			{"created_at": map[string]interface{}{"order": "desc"}},
		},
	}

	return query
}

// parseSearchResponse 解析 ES 搜索响应
func parseSearchResponse(res *esapi.Response) (*SearchResponse, error) {
	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source    json.RawMessage     `json:"_source"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read search response body failed: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal search response failed: %w", err)
	}

	resp := &SearchResponse{
		Total: result.Hits.Total.Value,
	}

	for _, hit := range result.Hits.Hits {
		var doc SearchPostDoc
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			zap.L().Warn("Failed to parse search hit", zap.Error(err))
			continue
		}

		if hl, ok := hit.Highlight["post_title"]; ok && len(hl) > 0 {
			doc.HighlightTitle = hl
		}
		if hl, ok := hit.Highlight["content"]; ok && len(hl) > 0 {
			doc.HighlightContent = hl
		}
		resp.Posts = append(resp.Posts, doc)
	}

	return resp, nil
}
