package sse_handler

import (
	"bluebell/internal/domain/entity"
	sse "bluebell/internal/infrastructure/sse"
	infratrace "bluebell/internal/infrastructure/trace"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

var tracer = infratrace.TracerForModule("handler/sse")

type Handler struct {
	hub *sse.Hub
}

func New(hub *sse.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) Events(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "SSEHandler.Events")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userIDVal, _ := c.Get("UserIDKey")
	userID, ok := userIDVal.(int64)
	if !ok {
		c.Status(401)
		return
	}

	_ = entity.UserID(userID)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ch := h.hub.Subscribe(userID)
	defer h.hub.Unsubscribe(userID)

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "data: %v\n\n", msg)
			return true
		case <-ctx.Done():
			return false
		}
	})
}

func (h *Handler) OnlineCount(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "SSEHandler.OnlineCount")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	c.JSON(200, gin.H{"online": h.hub.OnlineCount()})
}
