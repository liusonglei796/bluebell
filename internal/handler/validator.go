package handler

import (
	"bluebell/internal/dto/request"
	"fmt"
	"reflect"
	"strings"

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
func InitTrans(locale string) (err error) {
	if binding.Validator == nil {
		binding.Validator = &defaultValidator{validator: validator.New()}
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		zhT := zh.New()
		enT := en.New()

		uni := ut.New(enT, zhT, enT)

		var ok bool
		trans, ok = uni.GetTranslator(locale)
		if !ok {
			return fmt.Errorf("uni.GetTranslator(%s) failed", locale)
		}

		switch locale {
		case "en":
			err = en_translations.RegisterDefaultTranslations(v, trans)
		case "zh":
			err = zh_translations.RegisterDefaultTranslations(v, trans)
		default:
			err = en_translations.RegisterDefaultTranslations(v, trans)
		}

		v.RegisterStructValidation(SignUpParamStructLevelValidation, request.SignUpRequest{})
	}
	return
}

// removeTopStruct 去除提示信息中的结构体名称
func removeTopStruct(fields map[string]string) map[string]string {
	res := make(map[string]string)
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

// SignUpParamStructLevelValidation 自定义结构体验证函数
func SignUpParamStructLevelValidation(sl validator.StructLevel) {
	su := sl.Current().Interface().(request.SignUpRequest)
	if su.Password != su.RePassword {
		sl.ReportError(su.RePassword, "re_password", "RePassword", "eqfield", "password")
	}
}

// defaultValidator 实现 StructValidator 接口
type defaultValidator struct {
	validator *validator.Validate
}

func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	return v.validator.Struct(obj)
}

func (v *defaultValidator) Engine() interface{} {
	return v.validator
}
