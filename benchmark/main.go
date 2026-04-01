package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL    = "http://localhost:8080"
	token      = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiItMjIwMzA3OTk3Nzk4ODA5NiIsImV4cCI6MTc3NTAyMTQwOSwiaWF0IjoxNzc1MDE0MjA5fQ.pxvjp6-9dSxCjss1bPflXRZhJ95jJRCwruq8jkZ5c44"
	concurrent = 200
	totalReqs  = 2000
)

type VoteReq struct {
	PostID    int64 `json:"post_id"`
	Direction int8  `json:"direction"`
}

type VoteResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func main() {
	postIDs := []int64{-4370784420098048, -4370167882575872, 100}

	var (
		success int64
		failed  int64
		wg      sync.WaitGroup
		sem     = make(chan struct{}, concurrent)
	)

	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()

	for i := 0; i < totalReqs; i++ {
		sem <- struct{}{}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			postID := postIDs[idx%len(postIDs)]
			direction := int8(1)
			if idx%2 == 0 {
				direction = -1
			}

			body, _ := json.Marshal(VoteReq{PostID: postID, Direction: direction})
			req, _ := http.NewRequest("POST", baseURL+"/api/v1/vote", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := client.Do(req)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			var vResp VoteResp
			json.Unmarshal(respBody, &vResp)

			if vResp.Code == 1000 {
				atomic.AddInt64(&success, 1)
			} else {
				atomic.AddInt64(&failed, 1)
				if atomic.LoadInt64(&failed) <= 5 {
					fmt.Printf("  Failed response: code=%d msg=%s body=%s\n", vResp.Code, vResp.Msg, string(respBody))
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	s := atomic.LoadInt64(&success)
	f := atomic.LoadInt64(&failed)
	qps := float64(s+f) / elapsed.Seconds()

	fmt.Printf("=== Benchmark Results ===\n")
	fmt.Printf("Total Requests:  %d\n", totalReqs)
	fmt.Printf("Concurrency:     %d\n", concurrent)
	fmt.Printf("Success:         %d\n", s)
	fmt.Printf("Failed:          %d\n", f)
	fmt.Printf("Duration:        %.2fs\n", elapsed.Seconds())
	fmt.Printf("QPS:             %.2f\n", qps)
}
