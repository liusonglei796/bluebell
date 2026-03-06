package repointerface

import "errors"

// Domain 层错误定义
// 将原 dao/mysql 中的哨兵错误提升到 domain 层，使 service 层不再依赖具体的 DAO 包
var (
	ErrUserExist       = errors.New("用户已存在")
	ErrUserNotExist    = errors.New("用户不存在")
	ErrInvalidPassword = errors.New("密码错误")
	ErrVoteTimeExpire  = errors.New("投票时间已过")
	ErrVoteRepeated    = errors.New("不允许重复投票")
)
