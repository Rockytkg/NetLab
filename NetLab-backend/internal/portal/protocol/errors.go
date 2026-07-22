package protocol

import "errors"

var (
	ErrPacketTooShort       = errors.New("portal: 报文长度不足")
	ErrPacketTooLarge       = errors.New("portal: 报文长度超过限制")
	ErrUnsupportedVersion   = errors.New("portal: 不支持的协议版本")
	ErrInvalidReserved      = errors.New("portal: 保留字段非法")
	ErrInvalidUserPort      = errors.New("portal: 用户端口字段非法")
	ErrInvalidAttribute     = errors.New("portal: 属性字段非法")
	ErrAttributeCount       = errors.New("portal: 属性数量不匹配")
	ErrHandlerAlreadyExists = errors.New("portal: 协议处理器已注册")
	ErrHandlerNotFound      = errors.New("portal: 协议处理器未注册")
)
