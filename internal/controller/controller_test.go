package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bluebell/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleSuccess_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"foo": "bar"}
	HandleSuccess(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data map[string]string `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Msg)
	assert.Equal(t, "bar", resp.Data["foo"])
}

func TestHandleSuccess_NilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	HandleSuccess(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Msg)
	assert.Nil(t, resp.Data)
}

func TestHandleError_Classifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		err          error
		expectedCode int
		expectedHTTP int
	}{
		{model.ErrInvalidParam, CodeInvalidParam, http.StatusBadRequest},
		{model.ErrNotFound, CodeNotFound, http.StatusNotFound},
		{model.ErrNeedLogin, CodeNeedLogin, http.StatusUnauthorized},
		{model.ErrInvalidToken, CodeInvalidToken, http.StatusUnauthorized},
		{model.ErrForbidden, CodeForbidden, http.StatusForbidden},
		{model.ErrUserExist, CodeUserExist, http.StatusConflict},
		{model.ErrVoteRepeated, CodeVoteRepeated, http.StatusConflict},
		{model.ErrRateLimitExceeded, CodeRateLimitExceeded, http.StatusTooManyRequests},
		{model.ErrDuplicateSubmit, CodeDuplicateSubmit, http.StatusTooManyRequests},
		{model.ErrServerBusy, CodeServerBusy, http.StatusServiceUnavailable},
		{errors.New("unknown error"), CodeServerBusy, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		HandleError(c, tt.err)

		assert.Equal(t, tt.expectedHTTP, w.Code)

		var resp struct {
			Code  int    `json:"code"`
			Msg   string `json:"msg"`
			Error string `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, tt.expectedCode, resp.Code)
		assert.Equal(t, tt.err.Error(), resp.Msg)
		assert.Equal(t, tt.err.Error(), resp.Error)
	}
}
