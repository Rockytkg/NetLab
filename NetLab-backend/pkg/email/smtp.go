package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"netlab-backend/pkg/i18n"
)

// Sender 是发送邮件的接口。
type Sender interface {
	// SendVerificationCode 以给定语言环境发送验证码邮件。
	// locale 应为 "zh-CN" 或 "en-US"；不支持时回退到 en-US。
	SendVerificationCode(to, code, purpose, locale string) error
}

// smtpSettings 是 SMTP 连接参数。pkg/email 不依赖任何上层配置类型，
// 由调用方通过 NewSMTPSenderFrom 传入具体字段。
type smtpSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// SMTPSender 通过 SMTP（可选 TLS）发送邮件。
type SMTPSender struct {
	cfg smtpSettings
}

// NewSMTPSenderFrom 根据显式字段构建一个 SMTPSender。
// 供运行时热加载配置的调用方使用（可用性由调用方判断）。
func NewSMTPSenderFrom(host string, port int, username, password, from string, useTLS bool) *SMTPSender {
	return &SMTPSender{cfg: smtpSettings{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   useTLS,
	}}
}

// SendVerificationCode 发送一封本地化的验证码邮件。
// locale 参数用于选择主题和正文的语言。
func (s *SMTPSender) SendVerificationCode(to, code, purpose, locale string) error {
	// 为 i18n 查找规范化 locale
	if !i18n.Supported(locale) {
		locale = i18n.DefaultLocale
	}

	var subjectKey, titleKey, descKey, actionKey, expiryKey string

	switch purpose {
	case "register":
		subjectKey = "email.register.subject"
		titleKey = "email.register.title"
		descKey = "email.register.description"
		expiryKey = "email.register.expiry"
	case "reset-password":
		subjectKey = "email.reset_password.subject"
		titleKey = "email.reset_password.title"
		descKey = "email.reset_password.description"
		expiryKey = "email.reset_password.expiry"
	default:
		subjectKey = "email.verification.subject"
		titleKey = "email.verification.title"
		descKey = "email.verification.description"
		expiryKey = "email.verification.expiry"
	}

	// 根据用途解析动作文案
	actionKey = "email.verification.default_action"
	if purpose == "register" {
		actionKey = "email.verification.register_action"
	} else if purpose == "reset-password" {
		actionKey = "email.verification.reset_action"
	}

	subject := i18n.MustT(locale, subjectKey)
	title := i18n.MustT(locale, titleKey)
	description := i18n.MustT(locale, descKey)
	action := i18n.MustT(locale, actionKey)
	expiry := i18n.MustT(locale, expiryKey)
	footer := i18n.MustT(locale, "email.footer")
	brand := i18n.MustT(locale, "email.brand")

	// 使用本地化字符串构建 HTML 正文
	body := buildVerificationHTML(title, description, action, code, expiry, footer, brand)

	return s.send(to, subject, body)
}

// buildVerificationHTML 构建本地化的 HTML 邮件正文。
func buildVerificationHTML(title, description, action, code, expiry, footer, brand string) string {
	// 当模板需要时，将动作文案插入到描述中。
	fullDesc := strings.Replace(description, "{{.Action}}", action, 1)

	return fmt.Sprintf(verificationHTMLTemplate,
		brand,    // 预览头 / 报头品牌
		title,    // 标题
		fullDesc, // 描述
		code,     // 验证码
		expiry,   // 过期 / 安全提示
		footer,   // 自动邮件页脚
		brand,    // 页脚品牌行
	)
}

func (s *SMTPSender) send(to, subject, htmlBody string) error {
	headers := make(map[string]string)
	headers["From"] = s.cfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.UseTLS {
		return s.sendWithTLS(addr, to, msg.String())
	}
	return s.sendPlain(addr, to, msg.String())
}

func (s *SMTPSender) sendWithTLS(addr, to, message string) error {
	tlsConfig := &tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	return s.authAndSend(client, to, message)
}

func (s *SMTPSender) sendPlain(addr, to, message string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	return s.authAndSend(client, to, message)
}

func (s *SMTPSender) authAndSend(client *smtp.Client, to, message string) error {
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return nil
}

// verificationHTMLTemplate 是验证码邮件的 HTML 模板。
// 其视觉语言遵循 Ant Design v6 设计令牌：
//   - 主品牌色 #1677FF，选中态浅色 #E6F4FF
//   - 中性表面 #F5F5F5（页面）/ #FFFFFF（卡片）/ #FAFAFA（验证码面板）
//   - 文本 #1F1F1F（主要）/ #595959（次要）/ #BFBFBF（弱化）
//   - 卡片圆角 8px，控件圆角 6px，卡片内边距 24px，4px 间距栅格
//   - 卡片投影：0 1px 2px rgba(0,0,0,.05)、0 1px 6px -1px rgba(0,0,0,.03)
//
// fmt.Sprintf 占位符（按顺序）：brand（报头）、title、description、code、
// expiry、footer、brand（页脚）。字面百分号转义为 %%。
const verificationHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="color-scheme" content="light">
</head>
<body style="margin:0;padding:0;background-color:#f5f5f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,'Noto Sans',sans-serif;-webkit-font-smoothing:antialiased;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f5f5f5;">
<tr><td align="center" style="padding:40px 16px;">
<table role="presentation" width="480" cellpadding="0" cellspacing="0" style="width:480px;max-width:100%%;">

<!-- Masthead -->
<tr><td style="padding:0 4px 16px;">
<table role="presentation" cellpadding="0" cellspacing="0"><tr>
<td style="width:32px;height:32px;background:#1677ff;border-radius:6px;text-align:center;vertical-align:middle;">
<span style="color:#ffffff;font-size:16px;font-weight:600;line-height:32px;">N</span>
</td>
<td style="padding-left:12px;vertical-align:middle;">
<span style="color:#1f1f1f;font-size:16px;font-weight:600;line-height:24px;">%s</span>
</td>
</tr></table>
</td></tr>

<!-- Card -->
<tr><td style="background:#ffffff;border:1px solid #f0f0f0;border-radius:8px;box-shadow:0 1px 2px 0 rgba(0,0,0,0.05),0 1px 6px -1px rgba(0,0,0,0.03),0 2px 4px 0 rgba(0,0,0,0.03);">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0">

<tr><td style="padding:32px 32px 0;">
<h1 style="margin:0;color:#1f1f1f;font-size:24px;line-height:32px;font-weight:600;">%s</h1>
<p style="margin:12px 0 0;color:#595959;font-size:14px;line-height:22px;">%s</p>
</td></tr>

<!-- Code panel -->
<tr><td style="padding:24px 32px 0;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#fafafa;border:1px solid #f0f0f0;border-radius:8px;">
<tr><td align="center" style="padding:24px 16px;">
<span style="font-family:'SFMono-Regular',Consolas,'Liberation Mono',Menlo,Courier,monospace;font-size:36px;line-height:44px;font-weight:600;letter-spacing:10px;color:#1677ff;">%s</span>
</td></tr>
</table>
</td></tr>

<!-- Expiry / security note -->
<tr><td style="padding:16px 32px 0;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#e6f4ff;border-radius:8px;">
<tr><td style="padding:8px 12px;color:#595959;font-size:12px;line-height:20px;">%s</td></tr>
</table>
</td></tr>

<tr><td style="padding:24px 32px 32px;">
<div style="border-top:1px solid #f0f0f0;padding-top:16px;">
<p style="margin:0;color:#bfbfbf;font-size:12px;line-height:20px;">%s</p>
</div>
</td></tr>

</table>
</td></tr>

<!-- Footer brand -->
<tr><td align="center" style="padding:24px 4px 0;">
<p style="margin:0;color:#bfbfbf;font-size:12px;line-height:20px;">%s</p>
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>`
