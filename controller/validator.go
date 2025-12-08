package controller

import (
	"fmt"
	"reflect"
	"strings"
	"bluebell/models"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// 定义全局翻译器
var trans ut.Translator

// InitTrans 初始化翻译器
// locale 参数指定需要初始化的语言，例如 "zh" 或 "en"
// 为什么：validator 默认的错误提示是英文，为了提升用户体验，需要配置国际化翻译
func InitTrans(locale string) (err error) {

	// 确保 Validator 已初始化
	// 在 Gin v1.9+ 中 binding.Validator 可能为 nil，需要先初始化
	if binding.Validator == nil {
		binding.Validator = &defaultValidator{validator: validator.New()}
	}
	
	// 修改 gin 框架中的 Validator 引擎属性，实现自定制
	// binding.Validator.Engine() 返回的是 interface{}，需要断言为 *validator.Validate 类型
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {

		// 注册一个获取 json tag 的自定义方法
		// 默认情况下 validator 使用结构体字段名（如 RePassword），这里改为使用 json tag（如 re_password）
		// 为什么：前端传参使用的是 json 字段名，报错信息也应该对应 json 字段名，而不是 Go 结构体字段名
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			// 获取 json tag 的值，并处理可能存在的选项（如 "name,omitempty"）
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			// 如果 json tag 是 "-"，说明该字段被忽略，返回空字符串
			if name == "-" {
				return ""
			}
			// 返回 json tag 的名称
			return name
		})

		zhT := zh.New() // 初始化中文翻译器
		enT := en.New() // 初始化英文翻译器

		// 第一个参数是备用（fallback）的语言环境，当找不到匹配语言时使用该语言
		// 后面的参数是应该支持的语言环境（支持多个）
		// 这里设置英文为 fallback，同时支持中文和英文
		// uni: Universal Translator，通用翻译器，保存了所有支持的语言环境
		uni := ut.New(enT, zhT, enT)

		// locale 通常传 "zh"
		var ok bool
		// 根据传入的 locale 获取对应的翻译器
		trans, ok = uni.GetTranslator(locale)
		if !ok {
			// 如果获取失败，返回错误
			return fmt.Errorf("uni.GetTranslator(%s) failed", locale)
		}

		// 注册翻译器
		// 根据 locale 注册对应的默认翻译规则
		switch locale {
		case "en":
			// 注册英文翻译
			err = en_translations.RegisterDefaultTranslations(v, trans)
		case "zh":
			// 注册中文翻译
			err = zh_translations.RegisterDefaultTranslations(v, trans)
		default:
			// 默认注册英文翻译
			err = en_translations.RegisterDefaultTranslations(v, trans)
		}
		// 注册自定义结构体校验
		// 为什么：有些复杂的校验规则（如两次密码是否一致）无法通过简单的 tag 实现，需要自定义校验函数
		v.RegisterStructValidation(SignUpParamStructLevelValidation, models.ParamSignUp{})
	}
	return
}

// removeTopStruct 去除提示信息中的结构体名称
// 为什么：validator 返回的错误信息默认带有结构体名称（如 "ParamSignUp.Username"），前端不需要这个前缀
func removeTopStruct(fields map[string]string) map[string]string {
	res := make(map[string]string)
	for field, err := range fields {
		// 截取点号之后的部分
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

// SignUpParamStructLevelValidation 自定义结构体验证函数
// 为什么：实现跨字段的校验逻辑，例如校验密码和确认密码是否一致
func SignUpParamStructLevelValidation(sl validator.StructLevel) {
	// sl.Current() 获取当前正在校验的结构体
	// .Interface() 获取其原始值 (interface{})
	// .(models.ParamSignUp) 类型断言为我们的目标结构体
	su := sl.Current().Interface().(models.ParamSignUp)
	if su.Password != su.RePassword {
		// 报告一个错误
		// 参数: (违规的字段值, 违规字段的json名, 违规字段的struct名, 规则名, 规则参数)
		// 这里我们复用 eqfield 规则名，这样可以利用已有的翻译（或者你可以自定义规则名如 "password_mismatch"）
		sl.ReportError(su.RePassword, "re_password", "RePassword", "eqfield", "password")
	}
}

// defaultValidator 是一个实现了 StructValidator 接口的结构体
// 用于在 Gin v1.9+ 中初始化 binding.Validator
type defaultValidator struct {
	validator *validator.Validate
}

// ValidateStruct 实现 StructValidator 接口的 ValidateStruct 方法
func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	return v.validator.Struct(obj)
}

// Engine 实现 StructValidator 接口的 Engine 方法
func (v *defaultValidator) Engine() interface{} {
	return v.validator
}