package entity

// VoteDirection 投票方向常量
const (
	VoteUp     int8 = 1  // 赞成
	VoteDown   int8 = -1 // 反对
	VoteRevoke int8 = 0  // 取消投票
)

// scorePerVote 每一票对帖子分数的影响值
// 来源：Reddit 早期算法中，一票约等于 432 分（基于时间衰减模型）
const scorePerVote = 432

// Vote 投票领域实体
type Vote struct {
	PostID    int64
	UserID    int64
	Direction int8 // 1: 赞成, -1: 反对, 0: 取消
}

// Validate 校验投票方向是否合法
// 核心业务规则：只允许 1（赞成）、-1（反对）、0（取消）三种操作
func (v *Vote) Validate() error {
	if v.Direction != VoteUp && v.Direction != VoteDown && v.Direction != VoteRevoke {
		return ErrInvalidParam
	}
	return nil
}

// ScoreDelta 计算此投票对帖子分数的影响值
// 赞成 +432，反对 -432，取消 0
func (v *Vote) ScoreDelta() float64 {
	return float64(v.Direction) * scorePerVote
}
