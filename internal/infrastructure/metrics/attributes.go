package metrics

import "go.opentelemetry.io/otel/attribute"

// 自定义属性 Key
var (
	// AttributeErrorType 错误分类标签：validation, auth, not_found, conflict, system 等
	AttributeErrorType = attribute.Key("error.type")

	// AttributeHTTPMethod HTTP 方法标签
	AttributeHTTPMethod = attribute.Key("http.method")

	// AttributeHTTPStatus HTTP 状态码标签
	AttributeHTTPStatus = attribute.Key("http.status_code")
)
