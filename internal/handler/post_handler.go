package handler

import (
	"strconv"

	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	dtoResp "bluebell/internal/dto/response"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// CreatePostHandler 创建帖子
func (h *postHandlerStruct) CreatePostHandler(c *gin.Context) {
	// 4. 调用 Service 层处理业务逻辑
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := &request.CreatePostRequest{}
	// 1. 尝试绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(p); err != nil {
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误，返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	if _, err := h.postService.CreatePost(c.Request.Context(), p, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *postHandlerStruct) GetPostDetailHandler(c *gin.Context) {
	// 1. 尝试获取路径参数
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		// 3. 如果解析失败，返回参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	data, err := h.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *postHandlerStruct) GetPostListHandler(c *gin.Context) {
	p := &request.PostListRequest{
		Page:  1,
		Size:  10,
		Order: request.OrderTime,
	}

	// 1. 尝试从 QueryString 绑定数据
	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误，返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	var data []*dtoResp.PostDetailResponse
	var err error

	// 4. 调用 Service 层处理业务逻辑
	if p.CommunityID == 0 {
		data, err = h.postService.GetPostList(c.Request.Context(), p)
	} else {
		data, err = h.postService.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *postHandlerStruct) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	// 1. 尝试从路径中获取参数
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		// 3. 如果解析失败，返回参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	if err := h.postService.DeletePost(c.Request.Context(), postID, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, nil)
}
