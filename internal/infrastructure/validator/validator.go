package validator

import (
	"bluebell/pkg/errorx"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// 定义全局翻译器
var Trans ut.Translator

// InitTrans 初始化翻译器, 默认使用中文
func InitTrans() (err error) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册获取 json tag 的自定义方法
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		zhT := zh.New()
		uni := ut.New(zhT, zhT)

		var ok bool
		Trans, ok = uni.GetTranslator("zh")
		if !ok {
			return errorx.New(errorx.CodeInfraError, "uni.GetTranslator(\"zh\") failed")
		}

		// 注册中文翻译
		if err = zh_translations.RegisterDefaultTranslations(v, Trans); err != nil {
			return errorx.Wrap(err, errorx.CodeInfraError, "zh_translations.RegisterDefaultTranslations failed")
		}
	}
	return nil
}

// RemoveTopStruct 去除提示信息中的结构体名称
func RemoveTopStruct(fields map[string]string) map[string]string {
	res := make(map[string]string)
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}
