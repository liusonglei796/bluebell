# System Optimization Implementation Plan - Phase 2: High-Concurrency Vote Buffering

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a local memory buffer to aggregate votes over a 100ms window, reducing Redis write pressure from 10k+ individual calls to batch updates.

**Architecture:** Producer-Consumer pattern using Go channels and a worker goroutine. Requests are buffered in a map and flushed periodically to Redis.

**Tech Stack:** Go, Redis, Go Channels, sync.Map.

---

### Task 1: Create the Vote Buffer Service

**Files:**
- Create: `internal/infrastructure/mq/vote_buffer.go`
- Modify: `internal/infrastructure/mq/message.go`

- [ ] **Step 1: Define the VoteBuffer structure**

```go
package mq

import (
	"context"
	"sync"
	"time"
)

type VoteBuffer struct {
	mu          sync.Mutex
	votes       map[string]int // key: postID:userID, value: direction
	flushInterval time.Duration
	ch          chan *VoteMessage
}

func NewVoteBuffer(interval time.Duration) *VoteBuffer {
	return &VoteBuffer{
		votes:         make(map[string]int),
		flushInterval: interval,
		ch:            make(chan *VoteMessage, 10000),
	}
}
```

- [ ] **Step 2: Implement the buffering and flushing logic**

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/mq/vote_buffer.go
git commit -m "feat(perf): add vote buffer skeleton for local aggregation"
```

---

### Task 2: Integrate Buffer into PostService

**Files:**
- Modify: `internal/application/post/post_service.go`
- Modify: `internal/di/di.go`

- [ ] **Step 1: Inject VoteBuffer into PostService**

- [ ] **Step 2: Update VoteForPost to use Buffer instead of direct Redis call**

- [ ] **Step 3: Implement the background flusher that calls Redis Pipeline**

- [ ] **Step 4: Commit**

```bash
git add internal/application/post/post_service.go internal/di/di.go
git commit -m "feat(perf): integrate vote buffer into post service"
```

---

### Task 3: Load Testing & Verification

- [ ] **Step 1: Write a benchmark test for VoteForPost**
- [ ] **Step 2: Run benchmark with and without buffer**
- [ ] **Step 3: Verify data consistency in Redis/MySQL**
- [ ] **Step 4: Commit**

```bash
git add internal/application/post/post_service_test.go
git commit -m "test(perf): add benchmarks for voting buffer"
```
