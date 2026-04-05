package es

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"bluebell/pkg/errorx"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.uber.org/zap"
)

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Posts    []SearchPostDoc `json:"posts"`
}

// SearchPostDoc 搜索结果文档
type SearchPostDoc struct {
	PostID           string   `json:"post_id"`
	AuthorID         int64    `json:"author_id"`
	CommunityID      int64    `json:"community_id"`
	PostTitle        string   `json:"post_title"`
	Content          string   `json:"content"`
	Status           int8     `json:"status"`
	CreatedAt        string   `json:"created_at"`
	HighlightTitle   []string `json:"highlight_title,omitempty"`
	HighlightContent []string `json:"highlight_content,omitempty"`
}

// Search 执行全文搜索，支持高亮和分页
// Called by: handler/search_handler.go (SearchHandler 中 esClient.Search(ctx, req))
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 50 {
		req.PageSize = 20
	}

	query := buildSearchQuery(req)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "marshal search query failed")
	}

	zap.L().Info("ES search",
		zap.String("keyword", req.Keyword),
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
	)

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(IndexPost),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "ES search request failed")
	}
	defer res.Body.Close()

	if res.IsError() {
		respBody, _ := io.ReadAll(res.Body)
		zap.L().Error("ES search error response",
			zap.Int("status", res.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, errorx.Newf(errorx.CodeServerBusy, "ES search error: %s", string(respBody))
	}

	resp, err := parseSearchResponse(res)
	if err != nil {
		return nil, err
	}

	resp.Page = req.Page
	resp.PageSize = req.PageSize

	return resp, nil
}

// buildSearchQuery 构建 ES 搜索 DSL
// Called by: Search (line 52: buildSearchQuery(req))
func buildSearchQuery(req *SearchRequest) map[string]interface{} {
	from := (req.Page - 1) * req.PageSize

	query := map[string]interface{}{
		"from": from,
		"size": req.PageSize,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Keyword,
				"fields": []string{"post_title^2", "content"},
				"type":   "best_fields",
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
			"includes": []string{"post_id", "author_id", "community_id", "post_title", "content", "status", "created_at"},
		},
		"sort": []map[string]interface{}{
			{"created_at": map[string]interface{}{"order": "desc"}},
		},
	}

	return query
}

// parseSearchResponse 解析 ES 搜索响应
// Called by: Search (line 85: parseSearchResponse(res))
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
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "read search response body failed")
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "unmarshal search response failed")
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

		// 提取高亮结果
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
