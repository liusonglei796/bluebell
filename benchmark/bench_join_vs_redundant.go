package main

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type BenchResult struct {
	Name        string
	TotalReqs   int
	TotalTime   time.Duration
	QPS         float64
	AvgLatency  time.Duration
	P50         time.Duration
	P90         time.Duration
	P95         time.Duration
	P99         time.Duration
	Errors      int
}

func runBenchmark(name string, db *sql.DB, query string, concurrency int, totalReqs int) BenchResult {
	reqsPerWorker := totalReqs / concurrency
	durations := make([]time.Duration, 0, totalReqs)
	var mu sync.Mutex
	var errCount int

	startTotal := time.Now()
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			localDurations := make([]time.Duration, 0, reqsPerWorker)
			localErrs := 0

			for j := 0; j < reqsPerWorker; j++ {
				t0 := time.Now()
				rows, err := db.Query(query)
				if err != nil {
					localErrs++
					continue
				}
				for rows.Next() {
					var pId, pTitle, author, comm, content string
					var id, commId, vNum, score int64
					var status int
					var createdAt time.Time
					_ = rows.Scan(&id, &pId, &commId, &pTitle, &author, &comm, &content, &status, &vNum, &score, &createdAt)
				}
				_ = rows.Close()
				localDurations = append(localDurations, time.Since(t0))
			}

			mu.Lock()
			durations = append(durations, localDurations...)
			errCount += localErrs
			mu.Unlock()
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTotal)

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	avg := time.Duration(0)
	if len(durations) > 0 {
		avg = sum / time.Duration(len(durations))
	}

	p50 := time.Duration(0)
	p90 := time.Duration(0)
	p95 := time.Duration(0)
	p99 := time.Duration(0)
	if len(durations) > 0 {
		p50 = durations[len(durations)*50/100]
		p90 = durations[len(durations)*90/100]
		p95 = durations[len(durations)*95/100]
		p99 = durations[len(durations)*99/100]
	}

	qps := float64(len(durations)) / totalTime.Seconds()

	return BenchResult{
		Name:       name,
		TotalReqs:  len(durations),
		TotalTime:  totalTime,
		QPS:        qps,
		AvgLatency: avg,
		P50:        p50,
		P90:        p90,
		P95:        p95,
		P99:        p99,
		Errors:     errCount,
	}
}

func main() {
	dsn := "root:15939087780Ll@@tcp(127.0.0.1:3307)/bluebell?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(100)

	// Warmup
	fmt.Println("Warming up database connection...")
	for i := 0; i < 100; i++ {
		rows, _ := db.Query("SELECT id, post_id, community_id, post_title, author_name, community_name, content, status, vote_num, score, created_at FROM post WHERE status = 1 ORDER BY created_at DESC LIMIT 10")
		if rows != nil {
			rows.Close()
		}
	}

	concurrency := 50
	totalReqs := 10000

	fmt.Printf("\n=== 开始基准对比测试 (并发: %d, 请求总数: %d) ===\n\n", concurrency, totalReqs)

	// 1. 方案 A：单表反范式冗余
	qA_Shallow := `SELECT id, post_id, community_id, post_title, author_name, community_name, content, status, vote_num, score, created_at 
FROM post 
WHERE status = 1 
ORDER BY created_at DESC 
LIMIT 10 OFFSET 0`

	// 2. 方案 B：无中间表，直接主键/唯一索引 JOIN（post + user + community）
	qB_Shallow := `SELECT p.id, p.post_id, p.community_id, p.post_title, COALESCE(u.user_name, ''), COALESCE(c.community_name, ''), p.content, p.status, p.vote_num, p.score, p.created_at
FROM post p
LEFT JOIN user u ON p.author_id = u.user_id
LEFT JOIN community c ON p.community_id = c.id
WHERE p.status = 1
ORDER BY p.created_at DESC
LIMIT 10 OFFSET 0`

	// 3. 方案 C：带中间表 post_author 多表 JOIN
	qC_Shallow := `SELECT p.id, p.post_id, p.community_id, p.post_title, COALESCE(u.user_name, ''), COALESCE(c.community_name, ''), p.content, p.status, p.vote_num, p.score, p.created_at
FROM post p
LEFT JOIN post_author pa ON p.post_id = pa.post_id
LEFT JOIN user u ON pa.user_id = u.user_id
LEFT JOIN community c ON p.community_id = c.id
WHERE p.status = 1
ORDER BY p.created_at DESC
LIMIT 10 OFFSET 0`

	resA_Shallow := runBenchmark("方案 A [浅分页-单表冗余]", db, qA_Shallow, concurrency, totalReqs)
	resB_Shallow := runBenchmark("方案 B [浅分页-直连主键JOIN]", db, qB_Shallow, concurrency, totalReqs)
	resC_Shallow := runBenchmark("方案 C [浅分页-中间表JOIN]", db, qC_Shallow, concurrency, totalReqs)

	// 深分页 OFFSET 500
	qA_Deep := `SELECT id, post_id, community_id, post_title, author_name, community_name, content, status, vote_num, score, created_at 
FROM post 
WHERE status = 1 
ORDER BY created_at DESC 
LIMIT 10 OFFSET 500`

	qB_Deep := `SELECT p.id, p.post_id, p.community_id, p.post_title, COALESCE(u.user_name, ''), COALESCE(c.community_name, ''), p.content, p.status, p.vote_num, p.score, p.created_at
FROM post p
LEFT JOIN user u ON p.author_id = u.user_id
LEFT JOIN community c ON p.community_id = c.id
WHERE p.status = 1
ORDER BY p.created_at DESC
LIMIT 10 OFFSET 500`

	qC_Deep := `SELECT p.id, p.post_id, p.community_id, p.post_title, COALESCE(u.user_name, ''), COALESCE(c.community_name, ''), p.content, p.status, p.vote_num, p.score, p.created_at
FROM post p
LEFT JOIN post_author pa ON p.post_id = pa.post_id
LEFT JOIN user u ON pa.user_id = u.user_id
LEFT JOIN community c ON p.community_id = c.id
WHERE p.status = 1
ORDER BY p.created_at DESC
LIMIT 10 OFFSET 500`

	resA_Deep := runBenchmark("方案 A [深分页-单表冗余]", db, qA_Deep, concurrency, totalReqs)
	resB_Deep := runBenchmark("方案 B [深分页-直连主键JOIN]", db, qB_Deep, concurrency, totalReqs)
	resC_Deep := runBenchmark("方案 C [深分页-中间表JOIN]", db, qC_Deep, concurrency, totalReqs)

	// 打印汇总表格
	fmt.Println("==========================================================================================================")
	fmt.Printf("%-26s | %-8s | %-10s | %-10s | %-10s | %-10s | %-10s\n", "测试方案", "QPS", "平均耗时", "P50", "P90", "P95", "P99")
	fmt.Println("----------------------------------------------------------------------------------------------------------")
	printRow(resA_Shallow)
	printRow(resB_Shallow)
	printRow(resC_Shallow)
	fmt.Println("----------------------------------------------------------------------------------------------------------")
	printRow(resA_Deep)
	printRow(resB_Deep)
	printRow(resC_Deep)
	fmt.Println("==========================================================================================================")
}

func printRow(r BenchResult) {
	fmt.Printf("%-26s | %-8.1f | %-10v | %-10v | %-10v | %-10v | %-10v\n",
		r.Name, r.QPS, r.AvgLatency.Round(time.Microsecond), r.P50.Round(time.Microsecond),
		r.P90.Round(time.Microsecond), r.P95.Round(time.Microsecond), r.P99.Round(time.Microsecond))
}
