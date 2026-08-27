package mysql

import (
	"testing"

	"bluebell/internal/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPostDao_JoinsSQL(t *testing.T) {
	// 验证 GORM Joins 生成的单条 SQL 语句
	db, err := gorm.Open(mysql.New(mysql.Config{
		DriverName:                "mysql",
		DSN:                       "root:root@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Skipf("Skip if mysql driver cannot initialize dryrun: %v", err)
	}

	// 1. 测试单条查询生成的 SQL
	stmt1 := db.Session(&gorm.Session{DryRun: true}).
		Joins("Author").
		Joins("Community").
		Where("post.post_id = ?", 12345).
		Where("post.status = ?", model.PostStatusPublished).
		First(&model.Post{}).Statement

	sql1 := stmt1.SQL.String()
	t.Logf("GetPostByID SQL: %s", sql1)
	assert.Contains(t, sql1, "LEFT JOIN")
	assert.Contains(t, sql1, "Author")
	assert.Contains(t, sql1, "Community")

	// 2. 测试批量查询生成的 SQL
	stmt2 := db.Session(&gorm.Session{DryRun: true}).
		Joins("Author").
		Joins("Community").
		Where("post.post_id IN (?)", []string{"101", "102"}).
		Where("post.status = ?", model.PostStatusPublished).
		Find(&[]*model.Post{}).Statement

	sql2 := stmt2.SQL.String()
	t.Logf("GetPostListByIDs SQL: %s", sql2)
	assert.Contains(t, sql2, "LEFT JOIN")
	assert.Contains(t, sql2, "Author")
	assert.Contains(t, sql2, "Community")

	// 3. 验证反范式冗余优化后的单表批量查询 SQL (Plan 1: 0 JOIN)
	stmt3 := db.Session(&gorm.Session{DryRun: true}).
		Model(&model.Post{}).
		Where("post_id IN ?", []string{"101", "102"}).
		Where("status = ?", model.PostStatusPublished).
		Find(&[]*model.Post{}).Statement

	sql3 := stmt3.SQL.String()
	t.Logf("GetPostListByIDsSingleTable SQL: %s", sql3)
	assert.NotContains(t, sql3, "JOIN")
	assert.Contains(t, sql3, "post_id IN")
}

