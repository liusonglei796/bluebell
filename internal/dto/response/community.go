package response

import "time"

// CommunityResponse 返回给客户端的社区信息
type CommunityResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Introduction string    `json:"introduction,omitempty"`
	CreateTime   time.Time `json:"create_time"`
}


