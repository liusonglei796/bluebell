package ai_handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/domain/svcdomain"
	aireq "bluebell/internal/dto/request/ai"
	"bluebell/internal/infrastructure/translate"
	"bluebell/pkg/errorx"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handler struct {
	aiSvc svcdomain.AiSerive
}

func New(aiSvc svcdomain.AiSerive) *Handler {
	return &Handler{
		aiSvc: aiSvc,
	}
}

func (h *Handler) SummaryHandler(c *gin.Context) {
	var summaryreq aireq.RemarkSummaryReq
	if err := c.ShouldBindJSON(&summaryreq); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			transerrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, translate.RemoveTopStruct(transerrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}
	
	// 添加调试日志
	zap.L().Info("AI Summary Request", 
		zap.Int64("post_id", summaryreq.PostID),
		zap.Int("max_comments", summaryreq.MaxComments))
	
	data, err := h.aiSvc.RemarkSummary(c.Request.Context(), &summaryreq)
	if err != nil {
		backfront.HandleError(c, errorx.ErrServerBusy)
		return
	}
	backfront.ResponseSuccess(c, data)
}
