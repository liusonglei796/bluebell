## Service 层接口提取重构

### 概述
已将 Service 层实现提取为接口定义，存放在 `internal/domain/service` 包中，实现依赖反转原则（DIP）。

### 新增文件结构

```
internal/domain/service/
├── user_service.go        # 用户服务接口
├── post_service.go        # 帖子服务接口
├── community_service.go   # 社区服务接口
├── vote_service.go        # 投票服务接口
└── services.go            # Services 聚合接口
```

### 接口定义

#### 1. UserService 接口
```go
type UserService interface {
    SignUp(ctx context.Context, p *request.SignUpRequest) error
    Login(ctx context.Context, p *request.LoginRequest) (accessToken, refreshToken string, err error)
    RefreshToken(ctx context.Context, accessToken, refreshToken string) (newAccessToken, newRefreshToken string, err error)
}
```

#### 2. PostService 接口
```go
type PostService interface {
    CreatePost(ctx context.Context, p *request.CreatePostRequest, authorID int64) (postID int64, err error)
    GetPostByID(ctx context.Context, pid int64) (*response.PostDetailResponse, error)
    GetPostList(ctx context.Context, p *request.PostListRequest) ([]*response.PostDetailResponse, error)
    GetCommunityPostList(ctx context.Context, p *request.PostListRequest) ([]*response.PostDetailResponse, error)
    DeletePost(ctx context.Context, postID int64, userID int64) error
}
```

#### 3. CommunityService 接口
```go
type CommunityService interface {
    GetCommunityList(ctx context.Context) ([]*response.CommunityResponse, error)
    GetCommunityDetail(ctx context.Context, id int64) (*response.CommunityResponse, error)
}
```

#### 4. VoteService 接口
```go
type VoteService interface {
    VoteForPost(ctx context.Context, userID int64, p *request.VoteRequest) error
}
```

#### 5. Services 聚合接口
```go
type Services interface {
    GetUserService() UserService
    GetPostService() PostService
    GetCommunityService() CommunityService
    GetVoteService() VoteService
}
```

### 实现类位置
所有接口的实现类仍然保持在原有位置：
- `internal/service/user/user_service.go` - UserService 实现
- `internal/service/post/post_service.go` - PostService 实现
- `internal/service/community/community_service.go` - CommunityService 实现
- `internal/service/vote/vote_service.go` - VoteService 实现

### 改进点

#### 1. 依赖反转
- Handler 层现在依赖接口而非具体实现
- Service 实现可轻松替换（如用 Mock 进行单元测试）

#### 2. 降低耦合
- Handler 与 Service 之间解耦
- 变更 Service 实现不会影响 Handler 层

#### 3. 便于测试
- 可为每个接口创建 Mock 实现
- 支持更灵活的单元测试

#### 4. 清晰的契约
- 接口明确定义了每个 Service 的职责
- 便于团队协作和文档维护

### 使用示例

#### Handler 中使用接口
```go
type UserHandler struct {
    userService domain.service.UserService
}

func NewUserHandler(userService domain.service.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

func (h *UserHandler) SignUpHandler(c *gin.Context) {
    // 使用接口方法
    if err := h.userService.SignUp(ctx, p); err != nil {
        // 处理错误
    }
}
```

#### 单元测试中使用 Mock
```go
type MockUserService struct {
    mock.Mock
}

func (m *MockUserService) SignUp(ctx context.Context, p *request.SignUpRequest) error {
    args := m.Called(ctx, p)
    return args.Error(0)
}

func TestSignUpHandler(t *testing.T) {
    mockService := new(MockUserService)
    mockService.On("SignUp", mock.Anything, mock.Anything).Return(nil)
    
    handler := NewUserHandler(mockService)
    // 测试逻辑...
}
```

### 兼容性
所有现有代码无需修改即可编译通过，新的接口定义与现有实现完全兼容。
