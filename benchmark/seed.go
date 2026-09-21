// Package main 用于为压测快速预置大量账号、帖子和 Token 数据
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"bluebell/internal/config"
	mysqldao "bluebell/internal/dao/mysql"
	redisdao "bluebell/internal/dao/redis"
	"bluebell/internal/jwt"
	"bluebell/internal/model"
	"bluebell/internal/snowflake"

	"gorm.io/gorm"
)

type SeedOutput struct {
	UserCount int      `json:"user_count"`
	PostCount int      `json:"post_count"`
	Tokens    []string `json:"tokens"`
	PostIDs   []int64  `json:"post_ids"`
}

func main() {
	var confFile string
	var userCount int
	var postCount int
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	flag.IntVar(&userCount, "users", 100, "预置用户数量")
	flag.IntVar(&postCount, "posts", 50, "预置帖子数量")
	flag.Parse()

	cfg, err := config.Init(confFile)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}

	if err := snowflake.Init(cfg); err != nil {
		fmt.Printf("初始化 snowflake 失败: %v\n", err)
		return
	}

	db, err := mysqldao.Init(cfg)
	if err != nil {
		fmt.Printf("连接 MySQL 失败: %v\n", err)
		return
	}
	defer mysqldao.Close(db)

	rdb, err := redisdao.Init(cfg)
	if err != nil {
		fmt.Printf("连接 Redis 失败: %v\n", err)
		return
	}
	defer redisdao.Close(rdb)

	postCache, _ := redisdao.NewPostCacheWithRefresher(rdb)
	ctx := context.Background()

	fmt.Printf(">>> 正在为压测生成 %d 个用户与 %d 篇帖子...\n", userCount, postCount)

	tokens := make([]string, 0, userCount)
	userIDs := make([]int64, 0, userCount)

	hashedPwd, _ := model.HashPassword("password123")

	// 1. 批量创建用户并签发 Token
	for i := 1; i <= userCount; i++ {
		uid := snowflake.GenID()
		username := fmt.Sprintf("k6_user_%d_%d", time.Now().Unix(), i)

		user := &model.User{
			UserID:   uid,
			UserName: username,
			Passwd:   hashedPwd,
			Role:     model.RoleUser,
		}
		db.Session(&gorm.Session{SkipHooks: true}).Create(user)

		aToken, _, err := jwt.GenToken(cfg, uid, model.RoleUser)
		if err == nil {
			tokens = append(tokens, aToken)
			userIDs = append(userIDs, uid)
		}
	}
	fmt.Printf(">>> 成功创建 %d 个用户并签发 Token！\n", len(tokens))

	// 2. 批量创建帖子并预热 Redis 榜单
	postIDs := make([]int64, 0, postCount)
	for i := 1; i <= postCount; i++ {
		pid := snowflake.GenID()
		authorID := userIDs[i%len(userIDs)]
		communityID := int64(1) // 默认 Go 社区

		post := &model.Post{
			PostID:      pid,
			PostTitle:   fmt.Sprintf("k6 压测样本帖子 #%d", i),
			Content:     fmt.Sprintf("这是一篇用于 k6 高并发全链路压测的帖子内容，序号: %d", i),
			Authors:     []model.User{{UserID: authorID}},
			CommunityID: communityID,
			Status:      1,
		}
		post.ContentHash = post.ComputeContentHash()
		db.Create(post)

		// 预热 Redis 排行榜与元数据 Hash
		_ = postCache.CreatePost(ctx, pid, communityID)
		postIDs = append(postIDs, pid)
	}
	fmt.Printf(">>> 成功创建 %d 篇帖子并完成 Redis 榜单预热！\n", len(postIDs))

	// 3. 导出为 tokens.json 供压测脚本读取
	out := SeedOutput{
		UserCount: len(tokens),
		PostCount: len(postIDs),
		Tokens:    tokens,
		PostIDs:   postIDs,
	}
	outData, _ := json.MarshalIndent(out, "", "  ")
	_ = os.WriteFile("benchmark/tokens.json", outData, 0644)

	fmt.Println(">>> 压测数据预热完成！数据已保存至 benchmark/tokens.json")
}
