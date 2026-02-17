# Go 切片返回值最佳实践：何时避免返回 nil

## 问题描述

在 Go 中，**nil 切片** 和 **空切片**（`[]T{}` 或 `make([]T, 0)`）在行为上有所不同，特别是在 JSON 序列化时：

```go
var nilSlice []int           // nil
emptySlice := []int{}        // 空切片 []

// JSON 序列化结果
json.Marshal(nilSlice)   // → "null"
json.Marshal(emptySlice) // → "[]"
```

返回 `nil` 会导致 API 返回 `"null"`，可能使前端（JavaScript/TypeScript）解析失败或需要额外处理。

---

## 必须返回空切片的场景

### 1. **API 响应体中的列表字段**

当函数返回值最终会被序列化为 JSON 返回给客户端时，**必须**返回空切片：

```go
// ✅ 正确 - 列表查询始终返回数组
func GetUserList() ([]*User, error) {
    users := make([]*User, 0)  // 初始化为空切片
    db.Find(&users)
    return users, nil  // 即使为空，返回 [] 而非 null
}

// ❌ 错误 - 可能返回 null
func GetUserListBad() ([]*User, error) {
    var users []*User  // nil
    db.Find(&users)
    return users, nil  // 查询为空时返回 null
}
```

**为什么：** 前端期望数组类型，收到 `null` 会导致 `TypeError: null is not iterable`。

---

### 2. **批量查询函数（输入为 ID 列表）**

当输入为空或查询结果为空时，应返回空切片：

```go
// ✅ 正确
func GetUsersByIDs(ids []int64) ([]*User, error) {
    if len(ids) == 0 {
        return make([]*User, 0), nil  // 返回空切片
    }
    // ... 查询逻辑
}

// ❌ 错误
func GetUsersByIDs(ids []int64) ([]*User, error) {
    if len(ids) == 0 {
        return nil, nil  // 返回 nil，API 会返回 null
    }
    // ... 查询逻辑
}
```

**为什么：** 输入为空是正常业务场景，不应返回 `null` 让前端处理。

---

### 3. **多条件筛选的列表接口**

当查询条件可能导致无结果时：

```go
// ✅ 正确
func SearchPosts(filter *PostFilter) ([]*Post, error) {
    posts := make([]*Post, 0)
    query := db.Model(&Post{})
    
    if filter.Category != "" {
        query = query.Where("category = ?", filter.Category)
    }
    // ... 更多条件
    
    query.Find(&posts)
    return posts, nil  // 无匹配时返回 [] 而非 null
}
```

---

### 4. **关联数据的嵌套列表**

ORM 查询中的嵌套列表字段：

```go
// ✅ 正确
func GetPostWithComments(postID int64) (*PostDetail, error) {
    detail := &PostDetail{
        Comments: make([]*Comment, 0),  // 初始化为空
    }
    db.Where("post_id = ?", postID).Find(&detail.Comments)
    return detail, nil
}
```

**为什么：** 即使帖子没有评论，前端期望 `"comments": []` 而非 `"comments": null`。

---

## 可以返回 nil 的场景

### 1. **明确标识"无数据"的业务含义**

当需要区分"无权限查看"和"有权限但无数据"时：

```go
// ✅ 正确 - 业务上需要区分
func GetUserPrivateData(userID int64) (*UserPrivate, error) {
    if !hasPermission(currentUser, userID) {
        return nil, ErrNoPermission  // nil 表示无权限
    }
    // ...
}
```

---

### 2. **单条记录查询（使用指针返回值）**

查询单条记录时，nil 表示记录不存在：

```go
// ✅ 正确
func GetUserByID(id int64) (*User, error) {
    user := &User{}
    err := db.First(user, id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil  // 记录不存在，返回 nil
    }
    return user, nil
}
```

**注意：** 这里是 `*User`（指针），不是 `[]User`（切片），语义清晰。

---

### 3. **内部逻辑处理，不直接序列化**

当返回值仅在内部使用，不会序列化为 JSON：

```go
// ✅ 正确 - 内部使用
func getRelatedIDs(entityID int64) ([]int64, error) {
    // 某些条件下直接返回 nil，调用方会检查长度
    if !shouldQuery(entityID) {
        return nil, nil
    }
    // ...
}

// 调用方处理
ids, _ := getRelatedIDs(123)
if len(ids) == 0 {  // len(nil) == 0，安全
    return
}
```

---

## 快速检查清单

| 场景 | 返回值类型 | 空结果应返回 |
|------|-----------|-------------|
| API 列表接口 | `[]T` | `[]T{}` |
| 批量查询 | `[]T` | `[]T{}` |
| 嵌套列表字段 | `[]T` | `[]T{}` |
| 单条查询 | `*T` | `nil` |
| 内部工具函数 | `[]T` | `nil` 或 `[]T{}` 均可 |

---

## 最佳实践总结

```go
// 规则 1：API 响应的列表字段 → 必须初始化
func ListUsers() ([]*User, error) {
    users := make([]*User, 0)  // 强制初始化
    db.Find(&users)
    return users, nil
}

// 规则 2：输入校验提前返回 → 返回空切片
func GetByIDs(ids []int64) ([]*User, error) {
    if len(ids) == 0 {
        return make([]*User, 0), nil
    }
    // ...
}

// 规则 3：单条查询 → 可以返回 nil
func GetOne(id int64) (*User, error) {
    // 查询不到返回 nil
}
```

---

## 参考

- [Go 官方文档 - nil slices](https://go.dev/wiki/CodeReviewComments#declaring-empty-slices)
- [GORM 文档 - Query](https://gorm.io/docs/query.html)
