package controller

import (
	notifreq "bluebell/internal/dto/request/notification"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// NotificationController 通知中心控制器
type NotificationController struct {
	notifSvc *service.NotificationService
}

// NewNotificationController 创建通知控制器实例
func NewNotificationController(notifSvc *service.NotificationService) *NotificationController {
	return &NotificationController{notifSvc: notifSvc}
}

// GetNotificationListHandler 获取通知列表
func (h *NotificationController) GetNotificationListHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &notifreq.NotificationListRequest{
		Page: 1,
		Size: 20,
	}
	_ = c.ShouldBindQuery(p)

	ctx := c.Request.Context()
	data, err := h.notifSvc.GetNotificationList(ctx, userID.(int64), p)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetUnreadCountHandler 获取未读数
func (h *NotificationController) GetUnreadCountHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	ctx := c.Request.Context()
	data, err := h.notifSvc.GetUnreadCount(ctx, userID.(int64))
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// MarkReadHandler 标记已读
func (h *NotificationController) MarkReadHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &notifreq.MarkReadRequest{}
	_ = c.ShouldBindJSON(p)

	ctx := c.Request.Context()
	if err := h.notifSvc.MarkAsRead(ctx, userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}
