package errno

import "errors"

// 定义项目中使用的自定义错误
// 用于区分不同类型的错误，方便上层进行错误处理和响应

var (
	// ErrorInvalidID 无效的ID错误
	ErrorInvalidID = errors.New("无效的ID")
	
	// ErrorQueryFailed 数据库查询失败错误
	ErrorQueryFailed = errors.New("查询失败")
)