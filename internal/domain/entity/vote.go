package entity

// Vote 投票领域实体
type Vote struct {
	PostID    int64
	UserID    int64
	Direction int8 // 1: 赞成, -1: 反对, 0: 取消
}

// VoteDirection 投票方向常量
const (
	VoteUp   int8 = 1  // 赞成
	VoteDown int8 = -1 // 反对
	VoteRevoke int8 = 0 // 取消投票
)
