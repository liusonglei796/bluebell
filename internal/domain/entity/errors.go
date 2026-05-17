package entity

import "errors"

// wrappedError wraps a sentinel error with a cause, preserving the sentinel's
// Error() message for external consumers while allowing errors.Is/Unwrap to
// traverse the full chain for internal debugging.
type wrappedError struct {
	sentinel error
	cause    error
}

func (e *wrappedError) Error() string   { return e.sentinel.Error() }
func (e *wrappedError) Unwrap() error   { return e.cause }
func (e *wrappedError) Is(target error) bool {
	return target == e.sentinel
}

// Wrap wraps `err` with `sentinel`. The wrapped error's Error() returns the
// sentinel's message, errors.Is(wrapped, sentinel) returns true, and
// errors.Unwrap returns the original err. If err is nil, returns sentinel.
func Wrap(sentinel, err error) error {
	if err == nil {
		return sentinel
	}
	return &wrappedError{sentinel: sentinel, cause: err}
}

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
