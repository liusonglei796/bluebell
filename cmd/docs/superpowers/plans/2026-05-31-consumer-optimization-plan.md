# Consumer Optimization & Graceful Shutdown Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the double acknowledgement and silent string parsing bugs in RabbitMQ consumers, and implement graceful shutdown waiting for the consumers to stop before main exit.

**Architecture:** Modify consumer execution to manage Ack/Nack correctly in caller loops, handle errors on parsing msg strings, and use `sync.WaitGroup` with context cancellation to cleanly stop consumer goroutines before process termination.

**Tech Stack:** Go, RabbitMQ (amqp091-go), zap logger

---

### Task 1: Fix Double Ack/Nack Bug in ES Consumer

**Files:**
- Modify: `internal/infrastructure/mq/es_consumer.go:58-95`

- [ ] **Step 1: Update `handleDelivery` to delegate Ack/Nack and return parsing errors**
  Change the method to:
  ```go
  func (c *SyncConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) error {
  	// 从 Headers 提取 Trace 上下文
  	ctx = otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(d.Headers))

  	var msg SyncMessage
  	if err := json.Unmarshal(d.Body, &msg); err != nil {
  		return fmt.Errorf("es_consumer: 反序列化消息失败: %w", err)
  	}

  	switch msg.Action {
  	case "delete":
  		if err := c.client.DeleteDocument(ctx, es.IndexPost, msg.PostID); err != nil {
  			return fmt.Errorf("es_consumer: 删除ES文档失败 (post_id: %s): %w", msg.PostID, err)
  		}
  	default:
  		doc := map[string]interface{}{
  			"post_id":      msg.PostID,
  			"author_id":    msg.AuthorID,
  			"community_id": msg.CommunityID,
  			"post_title":   msg.PostTitle,
  			"content":      msg.Content,
  			"status":       msg.Status,
  			"created_at":   msg.CreatedAt,
  		}
  		body, err := json.Marshal(doc)
  		if err != nil {
  			return fmt.Errorf("序列化文档失败: %w", err)
  		}
  		if err := c.client.IndexDocument(ctx, es.IndexPost, msg.PostID, bytes.NewReader(body)); err != nil {
  			return fmt.Errorf("es_consumer: 索引ES文档失败 (post_id: %s): %w", msg.PostID, err)
  		}
  	}

  	return nil
  }
  ```

- [ ] **Step 2: Run verification**
  Run: `go build ./internal/infrastructure/mq`
  Expected: Compile successfully.

- [ ] **Step 3: Commit**
  ```bash
  git add internal/infrastructure/mq/es_consumer.go
  git commit -m "fix(mq): delegate ack/nack to outer loop in es_consumer"
  ```

---

### Task 2: Handle String Parsing Errors in Vote Consumer

**Files:**
- Modify: `internal/infrastructure/mq/vote_consumer.go:82-95`

- [ ] **Step 1: Check errors for `strconv.ParseInt` inside `handleDelivery`**
  Modify lines 82-84 of `internal/infrastructure/mq/vote_consumer.go` to handle error explicitly:
  ```go
  	userID, err := strconv.ParseInt(msg.UserID, 10, 64)
  	if err != nil {
  		return fmt.Errorf("vote_consumer: 无效的 UserID %q: %w", msg.UserID, err)
  	}
  	postID, err := strconv.ParseInt(msg.PostID, 10, 64)
  	if err != nil {
  		return fmt.Errorf("vote_consumer: 无效的 PostID %q: %w", msg.PostID, err)
  	}
  ```

- [ ] **Step 2: Run verification**
  Run: `go build ./internal/infrastructure/mq`
  Expected: Compile successfully.

- [ ] **Step 3: Commit**
  ```bash
  git add internal/infrastructure/mq/vote_consumer.go
  git commit -m "fix(mq): handle parsing errors in vote_consumer"
  ```

---

### Task 3: Implement Graceful Shutdown in Sync Consumer Main

**Files:**
- Modify: `cmd/consumer/sync/main.go`

- [ ] **Step 1: Import `"sync"` package**
  Add `"sync"` package to the imports:
  ```go
  import (
  	"context"
  	"flag"
  	"fmt"
  	"os"
  	"os/signal"
  	"sync"
  	"syscall"
  ```

- [ ] **Step 2: Use `sync.WaitGroup` to wait for consumer to stop on signal**
  Update the main function execution block:
  ```go
  	zap.L().Info("Starting Sync Consumer (ES)...")
  	// 12. 在独立的协程中启动消费者监听
  	var wg sync.WaitGroup
  	wg.Add(1)
  	go func() {
  		defer wg.Done()
  		if err := consumer.Start(ctx); err != nil {
  			if ctx.Err() != nil {
  				zap.L().Info("sync consumer stopped gracefully")
  			} else {
  				zap.L().Error("sync consumer exited with error", zap.Error(err))
  			}
  		}
  	}()

  	// 13. 优雅关机：监听操作系统退出信号 (Ctrl+C 或 kill)
  	quit := make(chan os.Signal, 1)
  	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
  	// 阻塞直到接收到退出信号
  	<-quit

  	zap.L().Info("Shutting down Sync Consumer, waiting for processing to complete...")
  	cancel()
  	wg.Wait()

  	zap.L().Info("Sync Consumer stopped. Closing resources...")
  ```

- [ ] **Step 3: Run verification**
  Run: `go build -o tmp_sync_consumer ./cmd/consumer/sync`
  Expected: Compile successfully.

- [ ] **Step 4: Commit**
  ```bash
  git add cmd/consumer/sync/main.go
  git commit -m "feat(sync-consumer): implement graceful shutdown waiting for worker exit"
  ```

---

### Task 4: Implement Graceful Shutdown in Vote Consumer Main

**Files:**
- Modify: `cmd/consumer/vote/main.go`

- [ ] **Step 1: Import `"sync"` package**
  Add `"sync"` package to the imports:
  ```go
  import (
  	"context"
  	"flag"
  	"fmt"
  	"os"
  	"os/signal"
  	"sync"
  	"syscall"
  ```

- [ ] **Step 2: Use `sync.WaitGroup` to wait for consumer to stop on signal**
  Update the main function execution block:
  ```go
  	zap.L().Info("Starting Vote Consumer...")
  	var wg sync.WaitGroup
  	wg.Add(1)
  	go func() {
  		defer wg.Done()
  		if err := consumer.Start(ctx); err != nil {
  			if ctx.Err() != nil {
  				zap.L().Info("vote consumer stopped gracefully")
  			} else {
  				zap.L().Error("vote consumer exited with error", zap.Error(err))
  			}
  		}
  	}()

  	quit := make(chan os.Signal, 1)
  	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
  	<-quit

  	zap.L().Info("Shutting down Vote Consumer, waiting for processing to complete...")
  	cancel()
  	wg.Wait()

  	zap.L().Info("Vote Consumer stopped. Closing resources...")
  ```

- [ ] **Step 3: Run verification**
  Run: `go build -o tmp_vote_consumer ./cmd/consumer/vote`
  Expected: Compile successfully.

- [ ] **Step 4: Commit**
  ```bash
  git add cmd/consumer/vote/main.go
  git commit -m "feat(vote-consumer): implement graceful shutdown waiting for worker exit"
  ```
