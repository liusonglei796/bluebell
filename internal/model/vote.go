package model

import "gorm.io/gorm"

// 投票方向常量
const (
	VoteUp     int8 = 1  // 赞成
	VoteDown   int8 = -1 // 反对
	VoteRevoke int8 = 0  // 取消投票
)

// scorePerVote 每一票对帖子分数的影响值
// 来源：Reddit 早期算法中，一票约等于 432 分（基于时间衰减模型）
const scorePerVote = 432

// Vote 投票数据模型
// 对应数据库 vote 表
type Vote struct {
	gorm.Model
	PostID    int64 `gorm:"column:post_id;not null;index:idx_post_user,unique"`
	UserID    int64 `gorm:"column:user_id;not null;index:idx_post_user,unique"`
	Direction int8  `gorm:"column:direction;not null"` // 1: 赞成, -1: 反对, 0: 取消
}

// TableName 自定义表名
func (Vote) TableName() string {
	return "vote"
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
