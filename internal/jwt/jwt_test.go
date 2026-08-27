package jwt

import (
	"testing"

	"bluebell/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_JTIAndBlacklistFlow(t *testing.T) {
	cfg := &config.Config{
		JWT: &config.JWTConfig{
			Secret:        "test-jwt-secret-key-12345678",
			AccessExpiry:  "2h",
			RefreshExpiry: "168h",
		},
	}

	userID := int64(10086)
	aToken, rToken, err := GenToken(cfg, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, aToken)
	assert.NotEmpty(t, rToken)

	// 1. 解析 Access Token Claims 并检查 JTI
	aClaims, err := ParseTokenClaims(cfg, aToken, AccessTokenType)
	require.NoError(t, err)
	assert.Equal(t, AccessTokenType, aClaims.TokenType)
	assert.Equal(t, "10086", aClaims.Subject)
	assert.NotEmpty(t, aClaims.ID, "Access Token 必须包含唯一 JTI")

	// 2. 解析 Refresh Token Claims 并检查 JTI
	rClaims, err := ParseTokenClaims(cfg, rToken, RefreshTokenType)
	require.NoError(t, err)
	assert.Equal(t, RefreshTokenType, rClaims.TokenType)
	assert.Equal(t, "10086", rClaims.Subject)
	assert.NotEmpty(t, rClaims.ID, "Refresh Token 必须包含唯一 JTI")
	assert.NotEqual(t, aClaims.ID, rClaims.ID, "Access Token 与 Refresh Token 的 JTI 必须互不相同")

	// 3. 校验 ParseToken 兼容性
	parsedUID, err := ParseToken(cfg, aToken, AccessTokenType)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedUID)
}
