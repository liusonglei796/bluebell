// Package jwt 提供 JWT 生成与解析工具
package jwt

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// mustParseDuration 解析时间字符串，失败时 panic（配置错误属于启动期致命错误）
func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic("jwt: invalid duration string: " + s)
	}
	return d
}

// ParseToken 解析并验证 Token，返回 userID (支持 Access Token 和 Refresh Token)
func ParseToken(cfg *config.JWTConfig, tokenString string) (userID int64, err error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errorx.New(errorx.CodeInvalidToken, "无效的签名方法")
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil {
		return 0, errorx.Wrap(err, errorx.CodeInvalidToken, "Token 解析失败")
	}
	if !token.Valid {
		return 0, errorx.New(errorx.CodeInvalidToken, "无效的Token")
	}

	if claims.Subject == "" {
		return 0, errorx.New(errorx.CodeInvalidToken, "无效的用户ID")
	}

	userID, err = strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errorx.New(errorx.CodeInvalidToken, "无效的用户ID")
	}
	return userID, nil
}

// GenToken 生成 Access Token 和 Refresh Token
func GenToken(cfg *config.JWTConfig, userID int64) (aToken, rToken string, err error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.AccessExpiry))),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "bluebell",
	}
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", "", errorx.Wrap(err, errorx.CodeInfraError, "生成 AccessToken 失败")
	}

	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.RefreshExpiry))),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "bluebell",
	}).SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", "", errorx.Wrap(err, errorx.CodeInfraError, "生成 RefreshToken 失败")
	}

	return aToken, rToken, nil
}
