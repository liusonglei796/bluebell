package constants

const (
	// 帖子相关
	PostMaxTitleLen   = 128   // 帖子标题最大长度
	PostMaxContentLen = 10000 // 帖子内容最大长度
	PostListMaxSize   = 100   // 帖子列表每页最大条数
	PostListDefault   = 10    // 帖子列表每页默认条数

	// 投票相关
	VoteExpireWeeks = 1   // 投票过期时间（周），超过此时间不允许投票
	ScorePerVote    = 432 // 每票对应的分数权重: 86400/200

	// 雪花算法
	SnowflakeStartTime = "2024-01-01" // 雪花算法起始时间
)
