package social_handler

import (
	"bluebell/internal/application"
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	socialService application.SocialService
}

func New(socialService application.SocialService) *Handler {
	return &Handler{
		socialService: socialService,
	}
}

func (h *Handler) GetProfileHandler(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	currentUserID := int64(0)
	val, exists := c.Get("UserIDKey")
	if exists {
		currentUserID = val.(int64)
	}

	profile, err := h.socialService.GetProfile(c.Request.Context(), userID, currentUserID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, profile)
}

func (h *Handler) FollowHandler(c *gin.Context) {
	followingIDStr := c.Param("id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	val, exists := c.Get("UserIDKey")
	if !exists {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}
	followerID := val.(int64)

	err = h.socialService.FollowUser(c.Request.Context(), followerID, followingID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

func (h *Handler) UnfollowHandler(c *gin.Context) {
	followingIDStr := c.Param("id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	val, exists := c.Get("UserIDKey")
	if !exists {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}
	followerID := val.(int64)

	err = h.socialService.UnfollowUser(c.Request.Context(), followerID, followingID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

func (h *Handler) GetActivitiesHandler(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)

	activities, err := h.socialService.GetActivities(c.Request.Context(), userID, page, size)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, activities)
}
