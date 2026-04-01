package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL    = "http://localhost:8080"
	token      = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiItMjIwMzA3OTk3Nzk4ODA5NiIsImV4cCI6MTc3NTAyMTYzOCwiaWF0IjoxNzc1MDE0NDM4fQ.sGCmOEc3tUNrdDp7NBYJ0YxeO9Pc-E-x_uJmlt8Sg00"
	concurrent = 200
	duration   = 10 * time.Second
)

type VoteReq struct {
	PostID    int64 `json:"post_id"`
	Direction int8  `json:"direction"`
}

type VoteResp struct {
	Code int `json:"code"`
}

func main() {
	// 使用多个帖子分散行锁竞争
	postIDs := []int64{-4370784420098048, -4370167882575872, 100, 999}

	var (
		total   int64
		success int64
		failed  int64
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrent)
		stop    = make(chan struct{})
	)

	client := &http.Client{Timeout: 10 * time.Second}

	// 预热：确保所有帖子在 Redis 中存在
	for _, pid := range postIDs {
		body, _ := json.Marshal(VoteReq{PostID: pid, Direction: 1})
		req, _ := http.NewRequest("POST", baseURL+"/api/v1/vote", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ := client.Do(req)
		if resp != nil {
			io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}

	fmt.Printf("Starting benchmark: %d concurrent, %v duration\n", concurrent, duration)

	start := time.Now()

	// 启动并发 goroutine
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
			for {
				select {
				case <-stop:
					return
				default:
				}

				sem <- struct{}{}
				pid := postIDs[rng.Intn(len(postIDs))]
				dir := int8(1)
				if rng.Intn(2) == 0 {
					dir = -1
				}

				body, _ := json.Marshal(VoteReq{PostID: pid, Direction: dir})
				req, _ := http.NewRequest("POST", baseURL+"/api/v1/vote", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)

				resp, err := client.Do(req)
				if err != nil {
					atomic.AddInt64(&failed, 1)
					atomic.AddInt64(&total, 1)
					<-sem
					continue
				}

				var vResp VoteResp
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				json.Unmarshal(respBody, &vResp)

				atomic.AddInt64(&total, 1)
				if vResp.Code == 1000 {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&failed, 1)
				}
				<-sem
			}
		}()
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	t := atomic.LoadInt64(&total)
	s := atomic.LoadInt64(&success)
	f := atomic.LoadInt64(&failed)
	qps := float64(t) / elapsed

	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Concurrency:     %d\n", concurrent)
	fmt.Printf("Duration:        %.2fs\n", elapsed)
	fmt.Printf("Total Requests:  %d\n", t)
	fmt.Printf("Success:         %d (%.1f%%)\n", s, float64(s)/float64(t)*100)
	fmt.Printf("Failed:          %d (%.1f%%)\n", f, float64(f)/float64(t)*100)
	fmt.Printf("QPS:             %.0f\n", qps)
}
