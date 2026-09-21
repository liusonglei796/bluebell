package middleware

import (
	"encoding/json"
	"fmt"

	"bluebell/pkg/sqltrace"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SQLTraceMiddleware 统计每个 HTTP 请求生命周期内的所有 SQL 耗时
func SQLTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracker := &sqltrace.Tracker{
			Traces: make([]sqltrace.Item, 0, 8),
		}
		// 注入 request context
		c.Request = c.Request.WithContext(sqltrace.WithTracker(c.Request.Context(), tracker))

		c.Next()

		count := tracker.Count()
		cost := tracker.TotalCost()

		// 响应头添加 SQL 统计
		c.Header("X-SQL-Cost", cost.String())
		c.Header("X-SQL-Count", fmt.Sprintf("%d", count))

		traces := tracker.GetTraces()
		if len(traces) > 0 {
			type simpleTrace struct {
				SQL  string `json:"sql"`
				Cost string `json:"cost"`
			}
			simpleList := make([]simpleTrace, 0, len(traces))
			for _, tr := range traces {
				simpleList = append(simpleList, simpleTrace{
					SQL:  tr.SQL,
					Cost: tr.Duration.String(),
				})
			}
			if bytes, err := json.Marshal(simpleList); err == nil {
				c.Header("X-SQL-Details", string(bytes))
			}

			zap.L().Info("api sql trace",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("sql_count", count),
				zap.Duration("sql_cost", cost),
			)
		}
	}
}
