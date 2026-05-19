// Package health 提供健康检查端点
package health

import (
	"context"
	"net/http"
	"time"

	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler 健康检查处理器
type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
	es  *es.Client
}

// New 创建 Handler 实例
func New(db *gorm.DB, rdb *redis.Client, es *es.Client) *Handler {
	return &Handler{
		db:  db,
		rdb: rdb,
		es:  es,
	}
}

// Healthz 存活检查
// @Summary      存活检查
// @Description  返回服务是否存活
// @Tags         系统
// @Success      200  {object}  map[string]string  "{"status": "ok"}"
// @Router       /healthz [get]
func (h *Handler) Healthz(c *gin.Context) {
	logger.WithContext(c.Request.Context()).Info("healthz check")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz 就绪检查 - 检查 MySQL、Redis、ES 连接
// @Summary      就绪检查
// @Description  检查 MySQL、Redis、Elasticsearch 连接状态
// @Tags         系统
// @Success      200  {object}  map[string]string  "{"status": "ok"}"
// @Failure      503  {object}  map[string]string  "{"status": "error", "component": "mysql/redis/elasticsearch"}"
// @Router       /readyz [get]
func (h *Handler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// 1. 检查 MySQL
	sqlDB, err := h.db.DB()
	if err != nil {
		logger.WithContext(ctx).Error("readyz mysql failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "component": "mysql"})
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		logger.WithContext(ctx).Error("readyz mysql ping failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "component": "mysql"})
		return
	}

	// 2. 检查 Redis
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		logger.WithContext(ctx).Error("readyz redis ping failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "component": "redis"})
		return
	}

	// 3. 检查 ES（如果配置了）
	if h.es != nil {
		res, err := h.es.ES().Ping()
		if err != nil {
			logger.WithContext(ctx).Error("readyz es ping failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "component": "elasticsearch"})
			return
		}
		defer res.Body.Close()
		if res.IsError() {
			logger.WithContext(ctx).Error("readyz es ping error response", zap.String("status", res.Status()))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "component": "elasticsearch"})
			return
		}
	}

	logger.WithContext(ctx).Info("readyz check passed")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
