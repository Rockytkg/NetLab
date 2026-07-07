// Package i18n 使用 go-i18n 提供服务端国际化能力。
// 它支持 zh-CN 和 en-US 语言环境，并使用 JSON 消息文件。
// 消息以逻辑标识符为键（例如 "success"、"error.1001"），
// 并根据 I18N 中间件设置的 locale 进行翻译。
package i18n

import (
	"encoding/json"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// DefaultLocale 是没有匹配到 locale 时的兜底值。
const DefaultLocale = "en-US"

var (
	bundle     *i18n.Bundle
	bundleOnce sync.Once
	localizers sync.Map
)

// Init 使用给定的消息文件路径初始化 i18n bundle。
// 必须在应用启动期间调用一次，且早于任何 T() 调用。
// messagePaths 应指向 go-i18n v2 格式的 JSON 文件：
//
//	[
//	  { "id": "success", "translation": "操作成功" },
//	  ...
//	]
func Init(messagePaths ...string) error {
	var initErr error
	bundleOnce.Do(func() {
		b := i18n.NewBundle(language.English)
		b.RegisterUnmarshalFunc("json", json.Unmarshal)

		for _, p := range messagePaths {
			if _, err := b.LoadMessageFile(p); err != nil {
				initErr = err
				return
			}
		}
		bundle = b
	})
	return initErr
}

// T 将消息 ID 翻译为给定 locale 的文案。
// 如果 bundle 未初始化或翻译缺失，则返回消息 ID 本身作为兜底。
func T(locale string, msgID string, templateData any) string {
	if bundle == nil {
		return msgID
	}
	localizer := getLocalizer(locale)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: templateData,
	})
	if err != nil {
		return msgID
	}
	return msg
}

// MustT 类似 T，但为便于使用而接受一个 map 作为模板数据。
func MustT(locale, msgID string) string {
	return T(locale, msgID, nil)
}

// getLocalizer 返回给定 locale 的已缓存 localizer。
func getLocalizer(locale string) *i18n.Localizer {
	if l, ok := localizers.Load(locale); ok {
		return l.(*i18n.Localizer)
	}
	// 对于 go-i18n v2，直接使用 locale 字符串创建 localizer
	l := i18n.NewLocalizer(bundle, locale)
	localizers.Store(locale, l)
	return l
}

// Supported 检查某个 locale 字符串是否受支持。
func Supported(locale string) bool {
	// 检查本应用使用的常见受支持 locale
	switch locale {
	case "zh-CN", "en-US":
		return true
	default:
		return false
	}
}
