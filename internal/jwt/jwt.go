// Package jwt 提供 JWT 生成与解析工具
package jwt

import (
	"bluebell/internal/config"
	"bluebell/internal/model"
	"bluebell/internal/snowflake"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 定义 token 类型
type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"
)

// CustomClaims 自定义 Claims 包含 token 类型
type CustomClaims struct {
	TokenType TokenType `json:"type"`
	jwt.RegisteredClaims
}

// mustParseDuration 解析时间字符串，失败时 panic（配置错误属于启动期致命错误）
func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic("jwt: invalid duration string: " + s)
	}
	return d
}

// ParseTokenClaims 解析并验证 Token，返回完整 Claims
func ParseTokenClaims(cfg *config.Config, tokenString string, expectedType TokenType) (*CustomClaims, error) {
	claims := new(CustomClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("token 解析失败: %w", err)
	}
	if !token.Valid {
		return nil, model.ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, model.ErrInvalidToken
	}

	if claims.Subject == "" {
		return nil, model.ErrInvalidToken
	}

	return claims, nil
}

// ParseToken 解析并验证 Token，返回 userID 并校验 token 类型
func ParseToken(cfg *config.Config, tokenString string, expectedType TokenType) (userID int64, err error) {
	claims, err := ParseTokenClaims(cfg, tokenString, expectedType)
	if err != nil {
		return 0, err
	}

	userID, err = strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, model.ErrInvalidToken
	}
	return userID, nil
}

// GenToken 生成 Access Token 和 Refresh Token（附带唯一 JTI）
func GenToken(cfg *config.Config, userID int64) (aToken, rToken string, err error) {
	aJTI := strconv.FormatInt(snowflake.GenID(), 10)
	if aJTI == "0" {
		aJTI = fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())
	}
	aClaims := CustomClaims{
		TokenType: AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        aJTI,
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.JWT.AccessExpiry))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, aClaims).SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", "", fmt.Errorf("生成 AccessToken 失败: %w", err)
	}

	rJTI := strconv.FormatInt(snowflake.GenID(), 10)
	if rJTI == "0" {
		rJTI = fmt.Sprintf("%d_%d_r", userID, time.Now().UnixNano())
	}
	rClaims := CustomClaims{
		TokenType: RefreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        rJTI,
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.JWT.RefreshExpiry))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, rClaims).SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", "", fmt.Errorf("生成 RefreshToken 失败: %w", err)
	}

	return aToken, rToken, nil
}
