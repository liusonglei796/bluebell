package user_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bluebell/internal/domain/svcdomain"
	userreq "bluebell/internal/dto/request/user"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
)

// MockUserService 实现了 svcdomain.UserService 接口
type MockUserService struct {
	SignUpFunc       func(ctx context.Context, p *userreq.SignUpRequest) error
	LoginFunc        func(ctx context.Context, p *userreq.LoginRequest) (accessToken, refreshToken string, err error)
	RefreshTokenFunc func(ctx context.Context, p *userreq.RefreshTokenRequest) (newAccessToken, newRefreshToken string, err error)
}

func (m *MockUserService) SignUp(ctx context.Context, p *userreq.SignUpRequest) error {
	if m.SignUpFunc != nil {
		return m.SignUpFunc(ctx, p)
	}
	return nil
}

func (m *MockUserService) Login(ctx context.Context, p *userreq.LoginRequest) (accessToken, refreshToken string, err error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, p)
	}
	return "", "", nil
}

func (m *MockUserService) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAccessToken, newRefreshToken string, err error) {
	if m.RefreshTokenFunc != nil {
		return m.RefreshTokenFunc(ctx, p)
	}
	return "", "", nil
}

// ensure MockUserService implements svcdomain.UserService at compile time
var _ svcdomain.UserService = (*MockUserService)(nil)

func TestSignUpHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSignUpFunc func(ctx context.Context, p *userreq.SignUpRequest) error
		expectedCode   int
		expectedBody   string // 期望响应中包含的字符串
	}{
		{
			name: "Success",
			requestBody: userreq.SignUpRequest{
				Username:   "testuser",
				Password:   "password123",
				RePassword: "password123",
			},
			mockSignUpFunc: func(ctx context.Context, p *userreq.SignUpRequest) error {
				return nil // 模拟服务层成功
			},
			expectedCode: http.StatusOK,
			expectedBody: `success`, // 通常 backfront.ResponseSuccess 会包含类似 "msg":"success" 这样的结构
		},
		{
			name: "Service Error",
			requestBody: userreq.SignUpRequest{
				Username:   "existinguser",
				Password:   "password123",
				RePassword: "password123",
			},
			mockSignUpFunc: func(ctx context.Context, p *userreq.SignUpRequest) error {
				return errorx.ErrUserExist // 模拟服务层错误
			},
			expectedCode: http.StatusOK,
			expectedBody: `用户名已存在`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserService{
				SignUpFunc: tt.mockSignUpFunc,
			}
			h := New(mockService)

			// 设置路由和上下文
			r := gin.Default()
			r.POST("/signup", h.SignUpHandler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected HTTP code %d, but got %d", tt.expectedCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, but got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		requestBody   interface{}
		mockLoginFunc func(ctx context.Context, p *userreq.LoginRequest) (string, string, error)
		expectedCode  int
		expectedBody  string
	}{
		{
			name: "Success",
			requestBody: userreq.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockLoginFunc: func(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
				return "mock_access_token", "mock_refresh_token", nil
			},
			expectedCode: http.StatusOK,
			expectedBody: `"access_token":"mock_access_token"`, // 检查响应体里是否正确包裹了返回数据
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockUserService{
				LoginFunc: tt.mockLoginFunc,
			}
			h := New(mockService)

			r := gin.Default()
			r.POST("/login", h.LoginHandler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected HTTP code %d, but got %d", tt.expectedCode, w.Code)
			}
			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, but got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
