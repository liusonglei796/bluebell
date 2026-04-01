package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := "http://127.0.0.1:8080/api/v1/community"
	concurrency := 50
	requests := 10000

	var success int64
	var fail int64
	var totalDurationNs int64

	var wg sync.WaitGroup
	channel := make(chan struct{}, concurrency)

	start := time.Now()

	for i := 0; i < requests; i++ {
		wg.Add(1)
		channel <- struct{}{}

		go func() {
			defer wg.Done()
			reqStart := time.Now()

			resp, err := http.Get(url)
			duration := time.Since(reqStart)

			atomic.AddInt64(&totalDurationNs, duration.Nanoseconds())

			if err != nil || resp.StatusCode != 200 {
				atomic.AddInt64(&fail, 1)
			} else {
				resp.Body.Close()
				atomic.AddInt64(&success, 1)
			}

			<-channel
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	qps := float64(requests) / elapsed.Seconds()
	avgRT := time.Duration(totalDurationNs / int64(requests))

	fmt.Printf("========================================\n")
	fmt.Printf("压测结果\n")
	fmt.Printf("========================================\n")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("并发数: %d\n", concurrency)
	fmt.Printf("总请求数: %d\n", requests)
	fmt.Printf("成功: %d\n", success)
	fmt.Printf("失败: %d\n", fail)
	fmt.Printf("总耗时: %.2fs\n", elapsed.Seconds())
	fmt.Printf("QPS: %.2f\n", qps)
	fmt.Printf("平均响应时间: %v\n", avgRT)
	fmt.Printf("========================================\n")
}
