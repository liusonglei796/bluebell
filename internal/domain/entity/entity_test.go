package entity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUser_IsAdmin(t *testing.T) {
	var u *User
	assert.False(t, u.IsAdmin())

	u = &User{Role: RoleAdmin}
	assert.True(t, u.IsAdmin())
	u.Role = RoleUser
	assert.False(t, u.IsAdmin())
}

func TestUser_IsValid(t *testing.T) {
	var u *User
	assert.False(t, u.IsValid())
	u = &User{UserName: ""}
	assert.False(t, u.IsValid())
	u.UserName = "test"
	assert.True(t, u.IsValid())
}

func TestHashPassword(t *testing.T) {
	pw := "123456"
	hash, err := HashPassword(pw)
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, CheckPassword(pw, hash))
	assert.False(t, CheckPassword("wrong", hash))

	_, err = HashPassword("")
	assert.Equal(t, ErrInvalidParam, err)
}

func TestUser_Password(t *testing.T) {
	password := "test123456"
	hashed, err := HashPassword(password)
	assert.Nil(t, err)
	assert.NotEqual(t, password, hashed)
	assert.True(t, CheckPassword(password, hashed))
	assert.False(t, CheckPassword("wrong", hashed))
}

func TestPost_CanBeDeletedBy(t *testing.T) {
	p := &Post{AuthorID: 123}
	assert.Nil(t, p.CanBeDeletedBy(123))
	assert.Equal(t, ErrForbidden, p.CanBeDeletedBy(456))
}

func TestPost_IsValid(t *testing.T) {
	var p *Post
	assert.False(t, p.IsValid())
	p = &Post{PostID: ""}
	assert.False(t, p.IsValid())
	p.PostID = "123"
	p.PostTitle = "test title"
	p.Content = "test content"
	assert.True(t, p.IsValid())
}

func TestVote_Validate(t *testing.T) {
	v := &Vote{Direction: VoteUp}
	assert.Nil(t, v.Validate())
	v.Direction = VoteDown
	assert.Nil(t, v.Validate())
	v.Direction = VoteRevoke
	assert.Nil(t, v.Validate())
	v.Direction = 2
	assert.Equal(t, ErrInvalidParam, v.Validate())
}

func TestVote_ScoreDelta(t *testing.T) {
	v := &Vote{Direction: VoteUp}
	assert.Equal(t, float64(432), v.ScoreDelta())
	v.Direction = VoteDown
	assert.Equal(t, float64(-432), v.ScoreDelta())
	v.Direction = VoteRevoke
	assert.Equal(t, float64(0), v.ScoreDelta())
}

func TestRemark_Validate(t *testing.T) {
	r := &Remark{Content: "test"}
	assert.Nil(t, r.Validate())
	r.Content = "  "
	assert.Equal(t, ErrInvalidParam, r.Validate())
}
