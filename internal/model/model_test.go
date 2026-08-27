package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestPost_CanBeDeletedBy(t *testing.T) {
	p := &Post{Authors: []User{{UserID: 123}}}
	assert.Nil(t, p.CanBeDeletedBy(123))
	assert.Equal(t, ErrForbidden, p.CanBeDeletedBy(456))
}

func TestPost_IsValid(t *testing.T) {
	var p *Post
	assert.False(t, p.IsValid())
	p = &Post{PostID: ""}
	assert.False(t, p.IsValid())
	p = &Post{PostID: "123", PostTitle: "Test Title", Content: "Test Content"}
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


func TestComment_Validate(t *testing.T) {
	c := &Comment{Content: "Hello world"}
	assert.Nil(t, c.Validate())
	assert.True(t, c.IsRoot())

	c.RootID = 100
	assert.False(t, c.IsRoot())

	emptyComment := &Comment{Content: "   "}
	assert.Equal(t, ErrInvalidParam, emptyComment.Validate())
}

func TestModel_TableNames(t *testing.T) {
	assert.Equal(t, "post_comment", Comment{}.TableName())
	assert.Equal(t, "user_relation", UserRelation{}.TableName())
	assert.Equal(t, "user_notification", UserNotification{}.TableName())
	assert.Equal(t, "bookmark_folder", BookmarkFolder{}.TableName())
	assert.Equal(t, "post_bookmark", PostBookmark{}.TableName())
	assert.Equal(t, "tag", Tag{}.TableName())
	assert.Equal(t, "post_tag", PostTag{}.TableName())
	assert.Equal(t, "processed_events", ProcessedEvent{}.TableName())
}
