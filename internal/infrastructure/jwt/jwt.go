// Package jwt 提供 JWT 生成与解析工具
package jwt

import (
	"bluebell/pkg/errorx"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret             string        // 签名密钥
	AccessTokenExpiry  time.Duration // Access Token 有效期
	RefreshTokenExpiry time.Duration // Refresh Token 有效期
}

// 全局配置，由 Init 函数初始化
var jwtConfig *JWTConfig

// 导出有效期供外部使用（如设置 Redis TTL）
var (
	AccessTokenExpireDuration  time.Duration
	RefreshTokenExpireDuration time.Duration
)

// Init 初始化 JWT 配置
func Init(secret string, accessExpiry, refreshExpiry string) error {
	accessDuration, err := time.ParseDuration(accessExpiry)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "解析 AccessToken 过期时间 %q 失败", accessExpiry)
	}

	refreshDuration, err := time.ParseDuration(refreshExpiry)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "解析 RefreshToken 过期时间 %q 失败", refreshExpiry)
	}

	jwtConfig = &JWTConfig{
		Secret:             secret,
		AccessTokenExpiry:  accessDuration,
		RefreshTokenExpiry: refreshDuration,
	}
	AccessTokenExpireDuration = jwtConfig.AccessTokenExpiry
	RefreshTokenExpireDuration = jwtConfig.RefreshTokenExpiry
	return nil
}

// UserClaims 自定义 JWT 声明
type UserClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenToken 生成访问令牌和刷新令牌
func GenToken(userID int64, username string) (aToken, rToken string, err error) {
	c := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtConfig.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bluebell",
		},
	}
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(jwtConfig.Secret))
	if err != nil {
		return "", "", errorx.Wrap(err, errorx.CodeInfraError, "生成 AccessToken 失败")
	}

	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(jwtConfig.RefreshTokenExpiry)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "bluebell",
	}).SignedString([]byte(jwtConfig.Secret))
	if err != nil {
		return "", "", errorx.Wrap(err, errorx.CodeInfraError, "生成 RefreshToken 失败")
	}

	return aToken, rToken, nil
}

// ParseToken 解析并验证 Token
func ParseToken(tokenString string) (*UserClaims, error) {
	var mc = new(UserClaims)
	token, err := jwt.ParseWithClaims(tokenString, mc, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errorx.New(errorx.CodeInvalidToken, "无效的签名方法")
		}
		return []byte(jwtConfig.Secret), nil
	})
	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeInvalidToken, "Token 解析失败")
	}
	if token.Valid {
		return mc, nil
	}
	return nil, errorx.New(errorx.CodeInvalidToken, "无效的Token")
}

// ValidateRefreshToken 验证刷新令牌，返回解析出的 userID
func ValidateRefreshToken(rTokenString string) (userID int64, err error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(rTokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errorx.New(errorx.CodeInvalidToken, "无效的签名方法")
		}
		return []byte(jwtConfig.Secret), nil
	})

	if err != nil || !token.Valid {
		return 0, errorx.New(errorx.CodeInvalidToken, "RefreshToken 无效")
	}

	userID, err = strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errorx.Wrap(err, errorx.CodeInvalidToken, "Token 数据异常")
	}

	return userID, nil
}
