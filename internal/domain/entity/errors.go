package entity

import "errors"

// 领域层通用错误（替代原有的 errorx）
var (
	ErrNotFound          = errors.New("entity not found")
	ErrInvalidOperation  = errors.New("invalid operation")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrDuplicate         = errors.New("entity already exists")

	// 业务级错误
	ErrInvalidParam      = errors.New("invalid parameters")
	ErrUserExist         = errors.New("user already exists")
	ErrUserNotExist      = errors.New("user does not exist")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrServerBusy        = errors.New("server is busy")
	ErrNeedLogin         = errors.New("need login")
	ErrInvalidToken      = errors.New("invalid token")
	ErrVoteTimeExpire    = errors.New("vote time expired")
	ErrVoteRepeated      = errors.New("vote repeated")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrForbidden         = errors.New("forbidden operation")
	ErrRequestTimeout    = errors.New("request timeout")
	ErrNotLogin          = errors.New("not logged in")
)
