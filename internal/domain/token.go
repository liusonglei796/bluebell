package domain

import "time"

// TokenService 令牌服务接口 (JWT等实现)
type TokenService interface {
	GenToken(userID int64) (aToken, rToken string, err error)
	ParseToken(tokenString string, expectedType string) (userID int64, err error)
	GetAccessExpiry() time.Duration
	GetRefreshExpiry() time.Duration
}
