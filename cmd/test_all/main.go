package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type TestResult struct {
	Index       int           `json:"index"`
	Name        string        `json:"name"`
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	Status      int           `json:"status"`
	Duration    time.Duration `json:"duration"`
	SQLCount    int           `json:"sql_count"`
	SQLCost     string        `json:"sql_cost"`
	SQLCostDur  time.Duration `json:"sql_cost_dur"`
	SQLDetails  []SQLDetail   `json:"sql_details"`
	Error       string        `json:"error,omitempty"`
}

type SQLDetail struct {
	SQL  string `json:"sql"`
	Cost string `json:"cost"`
}

var (
	baseURL = "http://127.0.0.1:8080"
	client  = &http.Client{Timeout: 10 * time.Second}
	results []TestResult
)

func doRequest(idx int, name, method, path string, body interface{}, token string) (int, map[string]interface{}, *TestResult) {
	fullURL := baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		r := &TestResult{Index: idx, Name: name, Method: method, URL: path, Error: err.Error()}
		results = append(results, *r)
		return 0, nil, r
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		r := &TestResult{Index: idx, Name: name, Method: method, URL: path, Duration: duration, Error: err.Error()}
		results = append(results, *r)
		return 0, nil, r
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respMap map[string]interface{}
	_ = json.Unmarshal(respBody, &respMap)

	sqlCostStr := resp.Header.Get("X-SQL-Cost")
	if sqlCostStr == "" {
		sqlCostStr = "0s"
	}
	sqlCountStr := resp.Header.Get("X-SQL-Count")
	sqlCount, _ := strconv.Atoi(sqlCountStr)
	sqlCostDur, _ := time.ParseDuration(sqlCostStr)

	var sqlDetails []SQLDetail
	sqlDetailsStr := resp.Header.Get("X-SQL-Details")
	if sqlDetailsStr != "" {
		_ = json.Unmarshal([]byte(sqlDetailsStr), &sqlDetails)
	}

	r := &TestResult{
		Index:      idx,
		Name:       name,
		Method:     method,
		URL:        path,
		Status:     resp.StatusCode,
		Duration:   duration,
		SQLCount:   sqlCount,
		SQLCost:    sqlCostStr,
		SQLCostDur: sqlCostDur,
		SQLDetails: sqlDetails,
	}
	results = append(results, *r)

	fmt.Printf("[%2d] %-6s %-32s -> Status: %d | HTTP耗时: %8s | SQL条数: %2d | SQL耗时: %8s\n",
		idx, method, path, resp.StatusCode, duration.Round(time.Microsecond), sqlCount, sqlCostStr)

	return resp.StatusCode, respMap, r
}

func main() {
	fmt.Println("==========================================================================================")
	fmt.Println("                      Bluebell 全接口自动化测试 & SQL 耗时统计")
	fmt.Println("==========================================================================================")

	ts := time.Now().UnixNano() % 100000
	user1Name := fmt.Sprintf("testuser_%d", ts)
	user2Name := fmt.Sprintf("testuser2_%d", ts)
	password := "12345678"

	// 1. 用户注册 User 1
	idx := 1
	doRequest(idx, "用户注册 (User1)", "POST", "/api/v1/signup", map[string]interface{}{
		"username":    user1Name,
		"password":    password,
		"re_password": password,
	}, "")

	// 2. 用户登录 User 1
	idx++
	_, loginResp1, _ := doRequest(idx, "用户登录 (User1)", "POST", "/api/v1/login", map[string]interface{}{
		"username": user1Name,
		"password": password,
	}, "")
	data1, _ := loginResp1["data"].(map[string]interface{})
	user1Token, _ := data1["access_token"].(string)
	user1RefreshToken, _ := data1["refresh_token"].(string)
	user1IDFloat, _ := data1["user_id"].(float64)
	user1ID := int64(user1IDFloat)

	// 3. 用户登录 Admin
	idx++
	_, loginRespAdmin, _ := doRequest(idx, "管理员登录 (admin)", "POST", "/api/v1/login", map[string]interface{}{
		"username": "admin",
		"password": "admin123",
	}, "")
	dataAdmin, _ := loginRespAdmin["data"].(map[string]interface{})
	adminToken, _ := dataAdmin["access_token"].(string)

	// 4. 刷新 Token
	idx++
	doRequest(idx, "刷新 Token", "POST", "/api/v1/refresh_token", map[string]interface{}{
		"refresh_token": user1RefreshToken,
	}, "")

	// 5. 获取社区列表
	idx++
	doRequest(idx, "获取社区列表", "GET", "/api/v1/community", nil, "")

	// 6. 获取社区详情
	idx++
	doRequest(idx, "获取社区详情 (id=1)", "GET", "/api/v1/community/1", nil, "")

	// 7. 管理员创建社区
	idx++
	commName := fmt.Sprintf("GoCommunity_%d", ts)
	doRequest(idx, "创建社区 (Admin)", "POST", "/api/v1/community", map[string]interface{}{
		"name":         commName,
		"introduction": "Go language discussion and sharing",
	}, adminToken)

	// 8. 创建社区标签
	idx++
	tagName := fmt.Sprintf("Golang_%d", ts)
	doRequest(idx, "创建社区标签", "POST", "/api/v1/tag", map[string]interface{}{
		"community_id": 1,
		"name":         tagName,
	}, user1Token)

	// 9. 获取社区标签列表
	idx++
	doRequest(idx, "获取社区标签列表", "GET", "/api/v1/tags?community_id=1", nil, "")

	// 10. 创建帖子
	idx++
	postTitle := fmt.Sprintf("Test Post Title %d", ts)
	_, postResp, _ := doRequest(idx, "创建帖子", "POST", "/api/v1/post", map[string]interface{}{
		"title":        postTitle,
		"content":      "This is a test post content for SQL latency benchmarking.",
		"community_id": 1,
		"tag_ids":      []int{},
	}, user1Token)

	var createdPostID int64
	if pData, ok := postResp["data"].(map[string]interface{}); ok {
		if pidStr, ok := pData["post_id"].(string); ok {
			createdPostID, _ = strconv.ParseInt(pidStr, 10, 64)
		} else if pidNum, ok := pData["post_id"].(float64); ok {
			createdPostID = int64(pidNum)
		}
	}

	// 11. 获取公开帖子列表 (按时间排序)
	idx++
	_, postListResp, _ := doRequest(idx, "获取帖子列表 (按时间)", "GET", "/api/v1/posts?page=1&size=10&order=time", nil, "")

	if createdPostID == 0 {
		if listData, ok := postListResp["data"].([]interface{}); ok && len(listData) > 0 {
			if firstPost, ok := listData[0].(map[string]interface{}); ok {
				switch idVal := firstPost["id"].(type) {
				case float64:
					createdPostID = int64(idVal)
				case string:
					createdPostID, _ = strconv.ParseInt(idVal, 10, 64)
				}
				if createdPostID == 0 {
					switch idVal := firstPost["post_id"].(type) {
					case float64:
						createdPostID = int64(idVal)
					case string:
						createdPostID, _ = strconv.ParseInt(idVal, 10, 64)
					}
				}
			}
		}
	}
	if createdPostID == 0 {
		createdPostID = 1
	}

	// 12. 获取公开帖子列表 (按热度分排序)
	idx++
	doRequest(idx, "获取帖子列表 (按热度)", "GET", "/api/v1/posts?page=1&size=10&order=score", nil, "")

	// 13. 获取帖子详情
	idx++
	doRequest(idx, fmt.Sprintf("获取帖子详情 (id=%d)", createdPostID), "GET", fmt.Sprintf("/api/v1/post/%d", createdPostID), nil, "")

	// 14. 置顶帖子
	idx++
	doRequest(idx, "置顶帖子 (Admin)", "POST", fmt.Sprintf("/api/v1/post/pin?community_id=1"), map[string]interface{}{
		"post_id":   createdPostID,
		"is_pinned": 1,
	}, adminToken)

	// 15. 帖子投票
	idx++
	doRequest(idx, "帖子投票 (+1 赞同)", "POST", "/api/v1/vote", map[string]interface{}{
		"post_id":   createdPostID,
		"direction": 1,
	}, user1Token)

	// 16. 创建根评论
	idx++
	_, commentResp, _ := doRequest(idx, "发表根评论", "POST", "/api/v1/comment", map[string]interface{}{
		"post_id":      createdPostID,
		"content":      "This is a root comment for testing.",
		"root_id":      0,
		"parent_id":    0,
		"reply_to_uid": 0,
	}, user1Token)

	var createdRootCommentID int64
	if cData, ok := commentResp["data"].(map[string]interface{}); ok {
		if cid, ok := cData["id"].(float64); ok {
			createdRootCommentID = int64(cid)
		} else if cidStr, ok := cData["id"].(string); ok {
			createdRootCommentID, _ = strconv.ParseInt(cidStr, 10, 64)
		}
	}
	if createdRootCommentID == 0 {
		createdRootCommentID = 1
	}

	// 17. 创建二级楼中楼回复
	idx++
	_, subCommentResp, _ := doRequest(idx, "发表二级楼中楼回复", "POST", "/api/v1/comment", map[string]interface{}{
		"post_id":      createdPostID,
		"content":      "This is a sub reply comment.",
		"root_id":      createdRootCommentID,
		"parent_id":    createdRootCommentID,
		"reply_to_uid": user1ID,
	}, user1Token)

	var createdSubCommentID int64
	if sData, ok := subCommentResp["data"].(map[string]interface{}); ok {
		if sid, ok := sData["id"].(float64); ok {
			createdSubCommentID = int64(sid)
		} else if sidStr, ok := sData["id"].(string); ok {
			createdSubCommentID, _ = strconv.ParseInt(sidStr, 10, 64)
		}
	}
	if createdSubCommentID == 0 {
		createdSubCommentID = 2
	}

	// 18. 获取评论列表
	idx++
	doRequest(idx, "获取帖子评论列表", "GET", fmt.Sprintf("/api/v1/comments?post_id=%d&page=1&size=10", createdPostID), nil, "")

	// 19. 获取楼中楼子回复列表
	idx++
	doRequest(idx, "获取楼中楼子回复列表", "GET", fmt.Sprintf("/api/v1/comment/replies?post_id=%d&root_id=%d&page=1&size=10", createdPostID, createdRootCommentID), nil, "")

	// 20. 注册并登录用户 2 (用于测试社交关注)
	idx++
	doRequest(idx, "注册用户 (User2)", "POST", "/api/v1/signup", map[string]interface{}{
		"username":    user2Name,
		"password":    password,
		"re_password": password,
	}, "")

	idx++
	_, loginResp2, _ := doRequest(idx, "登录用户 (User2)", "POST", "/api/v1/login", map[string]interface{}{
		"username": user2Name,
		"password": password,
	}, "")
	data2, _ := loginResp2["data"].(map[string]interface{})
	user2Token, _ := data2["access_token"].(string)
	user2IDFloat, _ := data2["user_id"].(float64)
	user2ID := int64(user2IDFloat)

	// 21. 关注用户 (User1 -> User2)
	idx++
	doRequest(idx, "关注用户 (User1 -> User2)", "POST", "/api/v1/user/follow", map[string]interface{}{
		"target_user_id": user2ID,
		"action":         1,
	}, user1Token)

	// 22. 获取关注列表
	idx++
	doRequest(idx, "获取关注列表 (User1 Following)", "GET", fmt.Sprintf("/api/v1/user/%d/following?page=1&size=10", user1ID), nil, "")

	// 23. 获取粉丝列表
	idx++
	doRequest(idx, "获取粉丝列表 (User2 Followers)", "GET", fmt.Sprintf("/api/v1/user/%d/followers?page=1&size=10", user2ID), nil, "")

	// 24. 获取通知列表
	idx++
	doRequest(idx, "获取通知列表 (User2)", "GET", "/api/v1/notifications?page=1&size=10", nil, user2Token)

	// 25. 获取未读通知数
	idx++
	doRequest(idx, "获取未读通知数 (User2)", "GET", "/api/v1/notifications/unread", nil, user2Token)

	// 26. 标记通知为已读
	idx++
	doRequest(idx, "标记通知已读 (User2)", "POST", "/api/v1/notifications/read", map[string]interface{}{
		"all": true,
	}, user2Token)

	// 27. 创建收藏夹
	idx++
	folderName := fmt.Sprintf("MyFavorites_%d", ts)
	_, folderResp, _ := doRequest(idx, "创建收藏夹", "POST", "/api/v1/bookmark/folder", map[string]interface{}{
		"name":      folderName,
		"is_public": 1,
	}, user1Token)

	var createdFolderID int64
	if fData, ok := folderResp["data"].(map[string]interface{}); ok {
		if fid, ok := fData["id"].(float64); ok {
			createdFolderID = int64(fid)
		} else if fidStr, ok := fData["id"].(string); ok {
			createdFolderID, _ = strconv.ParseInt(fidStr, 10, 64)
		}
	}

	// 28. 获取用户收藏夹列表
	idx++
	doRequest(idx, "获取用户收藏夹列表", "GET", "/api/v1/bookmark/folders", nil, user1Token)

	// 29. 添加帖子到收藏夹
	idx++
	doRequest(idx, "添加帖子到收藏夹", "POST", "/api/v1/bookmark", map[string]interface{}{
		"post_id":   createdPostID,
		"folder_id": createdFolderID,
	}, user1Token)

	// 30. 获取收藏夹内帖子列表
	idx++
	doRequest(idx, "获取收藏帖子列表", "GET", fmt.Sprintf("/api/v1/bookmarks?folder_id=%d&page=1&size=10", createdFolderID), nil, user1Token)

	// 31. 取消收藏
	idx++
	doRequest(idx, "取消收藏帖子", "DELETE", "/api/v1/bookmark", map[string]interface{}{
		"post_id":   createdPostID,
		"folder_id": createdFolderID,
	}, user1Token)

	// 32. 删除评论
	idx++
	doRequest(idx, fmt.Sprintf("删除评论 (id=%d)", createdSubCommentID), "DELETE", fmt.Sprintf("/api/v1/comment/%d", createdSubCommentID), nil, user1Token)

	// 33. 删除帖子
	idx++
	doRequest(idx, fmt.Sprintf("删除帖子 (id=%d)", createdPostID), "DELETE", fmt.Sprintf("/api/v1/post/%d", createdPostID), nil, user1Token)

	// 34. 用户登出
	idx++
	doRequest(idx, "用户登出 (User1)", "POST", "/api/v1/logout", nil, user1Token)

	// 保存 JSON 测试报告
	data, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile("benchmark/sql_benchmark_results.json", data, 0644)

	// 打印汇总表格
	fmt.Println("\n==========================================================================================")
	fmt.Println("                                测试结果汇总表")
	fmt.Println("==========================================================================================")
	fmt.Printf("| %-3s | %-24s | %-6s | %-36s | %-6s | %-9s | %-8s | %-10s |\n",
		"序号", "接口名称", "方法", "接口路径", "状态码", "HTTP耗时", "SQL条数", "SQL耗时")
	fmt.Println("|-----|--------------------------|--------|--------------------------------------|--------|-----------|----------|------------|")

	var totalSQLDuration time.Duration
	var totalSQLQueries int

	for _, r := range results {
		totalSQLDuration += r.SQLCostDur
		totalSQLQueries += r.SQLCount
		fmt.Printf("| %-3d | %-24s | %-6s | %-36s | %-6d | %-9s | %-8d | %-10s |\n",
			r.Index, r.Name, r.Method, r.URL, r.Status, r.Duration.Round(time.Microsecond), r.SQLCount, r.SQLCost)
	}

	fmt.Println("|-----|--------------------------|--------|--------------------------------------|--------|-----------|----------|------------|")
	fmt.Printf("总计测试接口: %d 个 | SQL 查询总条数: %d 条 | SQL 累计耗时: %s\n", len(results), totalSQLQueries, totalSQLDuration)
	fmt.Println("==========================================================================================")
}
