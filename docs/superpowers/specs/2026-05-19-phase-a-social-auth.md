# Phase A: Social Infrastructure & OAuth Design Spec

**Date:** 2026-05-19
**Status:** Draft
**Scope:** User Profiles, Activity Feed (Event-Driven), Follow System, GitHub OAuth Framework.

---

## 1. Data Models (GORM)

### UserProfile
Stores extended user information.
```go
type UserProfile struct {
    gorm.Model
    UserID    int64  `gorm:"column:user_id;uniqueIndex"`
    AvatarURL string `gorm:"column:avatar_url"`
    Bio       string `gorm:"column:bio;type:text"`
    GitHubID  string `gorm:"column:github_id;index"`
    GitHubURL string `gorm:"column:github_url"`
}
```

### Follow
Stores directed social relationships.
```go
type Follow struct {
    gorm.Model
    FollowerID  int64 `gorm:"column:follower_id;index:idx_follow,unique"`
    FollowingID int64 `gorm:"column:following_id;index:idx_follow,unique"`
}
```

### Activity
Stores flattened events for fast retrieval.
```go
type Activity struct {
    gorm.Model
    UserID      int64  `gorm:"column:user_id;index"`
    Type        string `gorm:"column:type"` // "post", "vote", "follow", "comment"
    TargetID    string `gorm:"column:target_id"`
    TargetName  string `gorm:"column:target_name"` // Title of post, name of community, etc.
    Description string `gorm:"column:description"`
}
```

---

## 2. API Specifications (v1)

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| GET | `/user/:id` | Get profile, stats (followers, posts), and connectivity status. | No |
| GET | `/user/:id/activities` | Get paged activity feed. | No |
| POST | `/follow/:id` | Follow a user. | Yes |
| DELETE | `/follow/:id` | Unfollow a user. | Yes |
| GET | `/auth/github/login` | Redirect to GitHub OAuth page. | No |
| GET | `/auth/github/callback`| Process GitHub code and log in/link account. | No |

---

## 3. Technical Architecture: Event-Driven Activities

1.  **Event Source**: Application services (PostService, UserService) publish messages to RabbitMQ `exchange.activity`.
2.  **Message Format**:
    ```json
    {
      "user_id": 123,
      "type": "post_created",
      "target_id": "456",
      "timestamp": 1716000000
    }
    ```
3.  **Consumer**: `ActivityConsumer` listens to `queue.activity`, enriches data (e.g., fetches post title), and writes to the `Activity` table.

---

## 4. GitHub OAuth (Framework)
- Use `golang.org/x/oauth2` with a GitHub provider.
- Implement a `State` check to prevent CSRF.
- Mock Callback: If environment variables are missing, the callback will return a success JSON with dummy user data.
