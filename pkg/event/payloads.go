package event

// Event Type Constants
const (
	EventTypePostPublished = "event.post.published"
	EventTypeCommentCreated = "event.comment.created"
	EventTypeVoteCast       = "event.vote.cast"
	EventTypeUserFollowed   = "event.user.followed"
)

// PostPublishedEvent is published when a user submits and publishes a new post.
type PostPublishedEvent struct {
	PostID      int64  `json:"post_id"`
	AuthorID    int64  `json:"author_id"`
	CommunityID int64  `json:"community_id"`
	Title       string `json:"title"`
}

// CommentCreatedEvent is published when a top-level comment or nested reply is created.
type CommentCreatedEvent struct {
	CommentID      int64  `json:"comment_id"`
	PostID         int64  `json:"post_id"`
	RootID         int64  `json:"root_id"`          // 0 if top-level comment, otherwise root comment ID
	ParentID       int64  `json:"parent_id"`        // 0 if top-level, otherwise immediate parent comment ID
	AuthorID       int64  `json:"author_id"`        // ID of the user creating the comment
	ReplyToUserID  int64  `json:"reply_to_user_id"` // 0 if root comment, otherwise author of target comment/post
	ContentPreview string `json:"content_preview"`  // Truncated preview (max 100 runes) for instant notification
}

// VoteCastEvent is published when a user upvotes, downvotes, or resets their vote on a post.
type VoteCastEvent struct {
	PostID            int64 `json:"post_id"`
	UserID            int64 `json:"user_id"`
	Direction         int8  `json:"direction"`          // 1: Upvote, -1: Downvote, 0: Cancel/Reset
	PreviousDirection int8  `json:"previous_direction"` // 1, -1, or 0
}

// UserFollowedEvent is published when a user follows or unfollows another user.
type UserFollowedEvent struct {
	FollowerID  int64  `json:"follower_id"`  // User performing the action
	FollowingID int64  `json:"following_id"` // Target user
	Action      string `json:"action"`        // "follow" | "unfollow"
}
