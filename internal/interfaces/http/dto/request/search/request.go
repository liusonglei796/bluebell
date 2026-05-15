package searchreq

// SearchRequest 搜索请求参数
type SearchRequest struct {
	Keyword  string `form:"keyword" binding:"required"`      // 搜索关键词
	Page     int    `form:"page,default=1"`                // 页码，默认1
	PageSize int    `form:"page_size,default=20"`           // 每页数量，默认20
}