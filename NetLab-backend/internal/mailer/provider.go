// Package mailer 提供一个热加载的邮件发送器。
//
// 它在每次发送时从运行时配置服务读取当前 SMTP 设置，从而让管理端对
// 邮件配置的修改无需重启即可生效。底层复用 pkg/email 的 SMTP 实现。
package mailer

import (
	"context"

	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/pkg/email"
)

// Provider 是一个基于运行时配置的邮件发送器。
type Provider struct {
	configService *sysconfig.Service
}

// NewProvider 创建一个基于运行时配置的邮件 Provider。
func NewProvider(configService *sysconfig.Service) *Provider {
	return &Provider{configService: configService}
}

// IsEnabled 报告当前 SMTP 配置是否可用于发信。
func (p *Provider) IsEnabled(ctx context.Context) bool {
	if p == nil || p.configService == nil {
		return false
	}
	cfg, err := p.configService.SMTP(ctx)
	if err != nil {
		return false
	}
	return cfg.IsConfigured()
}

// SendVerificationCode 使用当前 SMTP 配置发送本地化验证码邮件。
func (p *Provider) SendVerificationCode(ctx context.Context, to, code, purpose, locale string) error {
	cfg, err := p.configService.SMTP(ctx)
	if err != nil {
		return err
	}
	sender := email.NewSMTPSenderFrom(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.From, cfg.UseTLS)
	return sender.SendVerificationCode(to, code, purpose, locale)
}

// SendTestFromStored 使用当前持久化的 SMTP 配置向指定地址发送测试邮件，
// 用于管理端的“测试邮件配置”功能。
func (p *Provider) SendTestFromStored(ctx context.Context, to, locale string) error {
	cfg, err := p.configService.SMTP(ctx)
	if err != nil {
		return err
	}
	sender := email.NewSMTPSenderFrom(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.From, cfg.UseTLS)
	return sender.SendVerificationCode(to, "000000", "verification", locale)
}
