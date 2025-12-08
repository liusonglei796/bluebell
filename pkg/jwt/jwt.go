package jwt

import (
	"bluebell/dao/mysql"
	"bluebell/models"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const AccessTokenExpireDuration = time.Minute * 10
const RefreshTokenExpireDuration = time.Hour * 24 * 30

var MySecret = []byte("Lay不吃压力")

// GenToken 生成访问令牌和刷新令牌
func GenToken(userID int64, username string) (aToken, rToken string, err error) {
	// 创建 Access Token
	c := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpireDuration)),
			Issuer:    "bluebell",
		},
	}
	// 加密 Access Token
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(MySecret)
	if err != nil {
		return "", "", err
	}

	// 创建 Refresh Token
	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenExpireDuration)),
		Issuer:    "bluebell",
	}).SignedString(MySecret)
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*UserClaims, error) {
	var mc = new(UserClaims)
	token, err := jwt.ParseWithClaims(tokenString, mc, func(token *jwt.Token) (i interface{}, err error) {
		return MySecret, nil
	})
	if err != nil {
		return nil, err
	}
	if token.Valid {
		return mc, nil
	}
	return nil, errors.New("invalid token")
}

// ValidateRefreshToken 验证刷新令牌，并返回用户信息
func ValidateRefreshToken(rTokenString string) (user *models.User, err error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(rTokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return MySecret, nil
	})

	if err != nil || !token.Valid {
		return user, errors.New("refresh token 无效")
	}

	userID := claims.Subject
	bizUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return user, errors.New("token数据异常")
	}
	
	// 调用刚才写的查询函数
	user, err = mysql.GetUserByID(bizUserID)
	if err != nil {
		return user, errors.New("用户不存在")
	}
	
	return user, nil
}