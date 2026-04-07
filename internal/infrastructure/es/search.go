package es

import (
	"bytes"         // 提供字节缓冲区操作，用于构建 HTTP 请求体
	"context"       // 提供上下文支持，用于请求取消和超时控制
	"encoding/json" // 提供 JSON 序列化和反序列化
	"io"            // 提供 I/O 操作接口

	"bluebell/pkg/errorx" // 项目自定义错误处理包

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
	Status           int8     `json:"status"`                      // 帖子状态（1=已发布，-1=审核失败）
	CreatedAt        string   `json:"created_at"`                  // 创建时间（RFC3339 格式）
	HighlightTitle   []string `json:"highlight_title,omitempty"`   // 标题高亮片段（omitempty 表示为空时不返回）
	HighlightContent []string `json:"highlight_content,omitempty"` // 内容高亮片段（omitempty 表示为空时不返回）
}

// Search 执行全文搜索，支持高亮和分页
// 这是 ES 搜索功能的核心入口方法
//
// 参数说明：
//   - ctx: 上下文，用于控制请求超时和取消
//   - req: 搜索请求对象，包含关键词和分页参数
//
// 返回值：
//   - *SearchResponse: 搜索结果响应对象，包含总条数、分页信息和文档列表
//   - error: 错误信息，搜索失败时返回错误
//
// Called by: handler/search_handler.go (SearchHandler 中 esClient.Search(ctx, req))
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	// ====== 第一步：参数校验和默认值处理 ======

	// 校验页码：如果页码小于 1，强制设为 1（防止无效请求）
	if req.Page < 1 {
		req.Page = 1
	}

	// 校验每页条数：如果小于 1 或大于 50，强制设为 20（防止恶意请求或过度消耗资源）
	if req.PageSize < 1 || req.PageSize > 50 {
		req.PageSize = 20
	}

	// ====== 第二步：构建 ES 查询 DSL ======

	// 调用 buildSearchQuery 构建 Elasticsearch 查询语句（DSL）
	// 返回一个 map 对象，包含 multi_match 查询、高亮配置、排序等
	query := buildSearchQuery(req)

	// ====== 第三步：序列化查询 DSL 为 JSON ======

	// 将 map 序列化为 JSON 字节数组，因为 ES API 需要 JSON 格式的请求体
	body, err := json.Marshal(query)
	if err != nil {
		// 序列化失败，包装错误并返回（通常是代码 bug，不应该发生）
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "marshal search query failed")
	}

	// ====== 第四步：记录搜索日志 ======

	// 记录搜索请求日志，便于调试和监控
	zap.L().Info("ES search",
		zap.String("keyword", req.Keyword), // 搜索关键词
		zap.Int("page", req.Page),          // 页码
		zap.Int("page_size", req.PageSize), // 每页条数
	)

	// ====== 第五步：执行 ES 搜索请求 ======

	// 调用 ES 客户端的 Search 方法，传入查询参数
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),                // 传递上下文，支持超时和取消
		c.es.Search.WithIndex(IndexPost),            // 指定搜索的索引名称（"post"）
		c.es.Search.WithBody(bytes.NewReader(body)), // 传递 JSON 查询DSL
		c.es.Search.WithTrackTotalHits(true),        // 启用总命中数追踪（ES 7+ 默认不返回精确总数）
	)
	if err != nil {
		// 网络错误或 ES 服务异常，包装错误并返回
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "ES search request failed")
	}
	// 确保响应体在函数返回前被关闭，防止 HTTP 连接泄漏
	defer res.Body.Close()

	// ====== 第六步：检查 ES 响应状态 ======

	// 检查 HTTP 响应状态码是否为错误（4xx 或 5xx）
	if res.IsError() {
		// 读取错误响应体，获取 ES 返回的详细错误信息
		respBody, _ := io.ReadAll(res.Body)

		// 记录错误日志，包含状态码和错误详情
		zap.L().Error("ES search error response",
			zap.Int("status", res.StatusCode),    // HTTP 状态码（如 400, 500）
			zap.String("body", string(respBody)), // ES 返回的错误详情
		)

		// 返回错误，包含 ES 的详细错误信息
		return nil, errorx.Newf(errorx.CodeServerBusy, "ES search error: %s", string(respBody))
	}

	// ====== 第七步：解析 ES 响应 ======

	// 调用 parseSearchResponse 解析 ES 返回的 JSON 响应
	resp, err := parseSearchResponse(res)
	if err != nil {
		// 解析失败（如 JSON 格式异常、字段缺失等）
		return nil, err
	}

	// ====== 第八步：填充分页信息 ======

	// ES 响应中不包含请求的分页参数，需要手动填充
	resp.Page = req.Page         // 当前页码
	resp.PageSize = req.PageSize // 每页条数

	// ====== 第九步：返回搜索结果 ======

	return resp, nil
}

// buildSearchQuery 构建 ES 搜索 DSL（领域特定语言）
// 该方法负责将搜索请求转换为 Elasticsearch 的查询 JSON 格式
//
// 参数说明：
//   - req: 搜索请求对象，包含关键词和分页参数
//
// 返回值：
//   - map[string]interface{}: ES 查询 DSL 的 map 表示
//
// Called by: Search (line 52: buildSearchQuery(req))
func buildSearchQuery(req *SearchRequest) map[string]interface{} {
	// ====== 计算分页偏移量 ======
	// 公式：offset = (page - 1) * page_size
	// 例如：page=2, page_size=20 → offset=20（跳过前 20 条）
	from := (req.Page - 1) * req.PageSize

	// ====== 构建查询 DSL ======
	// ES 查询 DSL 是一个嵌套的 JSON 对象，使用 map 表示
	query := map[string]interface{}{
		// ====== 分页参数 ======
		"from": from,         // 跳过前 N 条文档（偏移量）
		"size": req.PageSize, // 返回的文档数量

		// ====== 查询条件 ======
		"query": map[string]interface{}{
			// multi_match：多字段匹配查询
			// 同时在多个字段中搜索关键词，并综合计算相关度分数
			"multi_match": map[string]interface{}{
				"query": req.Keyword, // 搜索关键词（如 "Go 语言"）
				"fields": []string{
					"post_title^2", // 标题字段，^2 表示权重 ×2（标题匹配更重要）
					"content",      // 内容字段，默认权重 1
				},
				"type": "best_fields", // 取最佳匹配字段的分数（而非平均）
			},
		},

		// ====== 高亮配置 ======
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				// 标题字段高亮
				"post_title": map[string]interface{}{
					"pre_tags":  []string{"<em class='highlight'>"}, // 高亮前缀标签
					"post_tags": []string{"</em>"},                  // 高亮后缀标签
					// 注意：标题通常较短，不设置 fragment_size，返回整个标题
				},

				// 内容字段高亮
				"content": map[string]interface{}{
					"pre_tags":            []string{"<em class='highlight'>"}, // 高亮前缀
					"post_tags":           []string{"</em>"},                  // 高亮后缀
					"fragment_size":       150,                                // 每个片段最大 150 字符
					"number_of_fragments": 3,                                  // 最多返回 3 个片段（避免过长）
				},
			},
		},

		// ====== 返回字段过滤 ======
		// _source：指定返回哪些字段，减少网络传输
		"_source": map[string]interface{}{
			"includes": []string{
				"post_id",      // 帖子 ID
				"author_id",    // 作者 ID
				"community_id", // 社区 ID
				"post_title",   // 标题
				"content",      // 内容
				"status",       // 状态
				"created_at",   // 创建时间
			},
		},

		// ====== 排序规则 ======
		"sort": []map[string]interface{}{
			// 按创建时间降序排列（最新的帖子排在前面）
			{"created_at": map[string]interface{}{"order": "desc"}},
		},
	}

	// 返回构建好的查询 DSL
	return query
}

// parseSearchResponse 解析 ES 搜索响应
// 将 ES 返回的原始 JSON 解析为结构化的 SearchResponse 对象
//
// 参数说明：
//   - res: ES API 返回的 HTTP 响应对象
//
// 返回值：
//   - *SearchResponse: 解析后的搜索结果
//   - error: 解析失败时返回错误
//
// Called by: Search (line 85: parseSearchResponse(res))
func parseSearchResponse(res *esapi.Response) (*SearchResponse, error) {
	// ====== 定义响应结构体 ======
	// 使用匿名结构体来匹配 ES 返回的 JSON 格式
	// ES 响应格式：
	// {
	//   "hits": {
	//     "total": { "value": 123 },
	//     "hits": [
	//       {
	//         "_source": { ...文档数据... },
	//         "highlight": { "post_title": [...], "content": [...] }
	//       }
	//     ]
	//   }
	// }
	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"` // 总命中数
			} `json:"total"`
			Hits []struct {
				Source    json.RawMessage     `json:"_source"`   // 原始文档数据（JSON 字节）
				Highlight map[string][]string `json:"highlight"` // 高亮数据（字段名 → 高亮片段列表）
			} `json:"hits"`
		} `json:"hits"`
	}

	// ====== 读取响应体 ======
	// 从 HTTP 响应流中读取完整的 JSON 数据
	body, err := io.ReadAll(res.Body)
	if err != nil {
		// 读取失败（如网络中断、连接关闭）
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "read search response body failed")
	}

	// ====== 反序列化 JSON ======
	// 将 JSON 字节数组解析为 Go 结构体
	if err := json.Unmarshal(body, &result); err != nil {
		// JSON 格式错误或字段类型不匹配
		return nil, errorx.Wrapf(err, errorx.CodeServerBusy, "unmarshal search response failed")
	}

	// ====== 初始化响应对象 ======
	// 创建返回结果对象，填充总命中数
	resp := &SearchResponse{
		Total: result.Hits.Total.Value,
	}

	// ====== 遍历并解析每个搜索结果 ======
	// 将 ES 返回的 hits 转换为我们的 SearchPostDoc 格式
	for _, hit := range result.Hits.Hits {
		// 创建文档对象
		var doc SearchPostDoc

		// 反序列化 _source 字段为文档结构体
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			// 解析失败时记录警告日志，但不中断整个流程
			zap.L().Warn("Failed to parse search hit", zap.Error(err))
			continue // 跳过这条文档，继续处理下一条
		}

		// ====== 提取高亮结果 ======
		// ES 的高亮格式：highlight: { "post_title": ["..."], "content": ["..."] }

		// 提取标题高亮
		if hl, ok := hit.Highlight["post_title"]; ok && len(hl) > 0 {
			// 如果存在标题高亮片段，赋值给文档字段
			doc.HighlightTitle = hl
		}

		// 提取内容高亮
		if hl, ok := hit.Highlight["content"]; ok && len(hl) > 0 {
			// 如果存在内容高亮片段，赋值给文档字段
			doc.HighlightContent = hl
		}

		// 将处理好的文档添加到结果列表中
		resp.Posts = append(resp.Posts, doc)
	}

	// 返回解析完成的搜索结果
	return resp, nil
}
