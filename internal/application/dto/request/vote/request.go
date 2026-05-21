package votereq

// LeaderboardRequest 排行榜请求参数
type LeaderboardRequest struct {
	Size        int64 `form:"size" binding:"max=100"`   // 获取数量，最大100
	CommunityID int64 `form:"community_id"`             // 社区ID（可选，不传则获取全站排行榜）
}

// 排行榜类型常量
const (
	LeaderboardTypeGlobal    = "global"    // 全站排行榜
	LeaderboardTypeCommunity = "community" // 社区排行榜
)
