# 第17章:Redis在帖子排序中的应用

> **本章导读**
>
> 投票如何实时更新分数、帖子如何按分数排序？本章将讲解如何利用 Redis ZSet 实现高性能的帖子排序。

---

## 1. Redis 数据结构设计

### 1.1 ZSet 应用

- **按热度排序**: `bluebell:post:score` (Member: PostID, Score: Score)
- **按时间排序**: `bluebell:post:time` (Member: PostID, Score: Timestamp)
- **用户投票记录**: `bluebell:post:voted:{post_id}` (Member: UserID, Score: Direction)

---

## 2. Repository 接口实现

#### 文件: `internal/dao/redis/vote.go`

```go
// PostVote 为帖子投票
func (c *VoteCache) PostVote(ctx context.Context, userID int64, postID string, value float64) error {
	// 1. 获取帖子发布时间 (用于判断 7 天限制)
	postTime := c.rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()
	if float64(time.Now().Unix())-postTime > oneWeekInSeconds {
		return errorx.ErrVoteTimeExpire
	}

	// 2. 更新分数和记录 (使用 TxPipeline)
	pipeline := c.rdb.TxPipeline()
	pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), op*diff*scorePerVote, postID)
	// ... 更新投票记录 ...
	_, err := pipeline.Exec(ctx)
	return err
}
```

---

## 3. 批量查询优化 (Pipeline)

为了避免 N+1 问题，在获取帖子列表时，我们批量获取所有帖子的投票数。

```go
func (c *VoteCache) GetPostVoteData(ctx context.Context, ids []string) (data []int64, err error) {
	pipeline := c.rdb.Pipeline()
	for _, id := range ids {
		pipeline.ZCount(ctx, getRedisKey(KeyPostVotedZSetPrefix+id), "1", "1")
	}
	// ... 解析结果 ...
}
```

---

**下一章:** [第18章:按社区筛选帖子实现](./18-按社区筛选帖子实现.md)

**返回目录:** [README.md](./README.md)
