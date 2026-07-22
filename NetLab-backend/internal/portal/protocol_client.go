package portal

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/portal/protocol"
)

// ProtocolClient exchanges packets through an explicitly registered codec.
// It keeps vendor-specific framing and authenticator validation out of services.
type ProtocolClient struct {
	registry *protocol.Registry
	timeout  time.Duration
	port     int
	logger   *zap.Logger
}

// NewProtocolClient creates a protocol client with the standard NAS timeout.
func NewProtocolClient(registry *protocol.Registry, logger *zap.Logger) *ProtocolClient {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ProtocolClient{registry: registry, timeout: protocolTimeout, port: NASPort, logger: logger}
}

// Exchange sends one request and validates that its response matches the request.
func (c *ProtocolClient) Exchange(ctx context.Context, profile protocol.Profile, host string, port int, secret string, request *protocol.Packet) (*protocol.Packet, error) {
	handler, err := c.handler(profile)
	if err != nil {
		return nil, err
	}
	codec := handler.Codec()
	raw, err := codec.Encode(request, protocol.AuthContext{SharedSecret: []byte(secret)})
	if err != nil {
		return nil, fmt.Errorf("编码Portal请求: %w", err)
	}
	sent, err := codec.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("解析Portal请求: %w", err)
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(c.nasPort(port))))
	if err != nil {
		return nil, fmt.Errorf("解析Portal NAS地址: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("连接Portal NAS: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("设置Portal超时: %w", err)
	}
	if _, err := conn.Write(raw); err != nil {
		return nil, fmt.Errorf("发送Portal请求: %w", err)
	}
	buf := make([]byte, protocol.MaxPacketSize)
	n, err := conn.Read(buf)
	if err != nil {
		c.logger.Warn("Portal请求超时或读取失败", zap.String("host", host), zap.String("profile", string(profile)), zap.Error(err))
		return nil, fmt.Errorf("读取Portal响应: %w", err)
	}
	response, err := codec.Decode(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("解析Portal响应: %w", err)
	}
	if response.SerialNo != request.SerialNo || response.UserIP != request.UserIP {
		return nil, errorsMismatchedResponse
	}
	if err := codec.Verify(response, protocol.AuthContext{SharedSecret: []byte(secret), RequestAuthenticator: &sent.Authenticator}); err != nil {
		return nil, fmt.Errorf("校验Portal响应: %w", err)
	}
	return response, nil
}

// Send writes a one-way Portal acknowledgement packet.
func (c *ProtocolClient) Send(ctx context.Context, profile protocol.Profile, host string, port int, secret string, packet *protocol.Packet) error {
	handler, err := c.handler(profile)
	if err != nil {
		return fmt.Errorf("获取Portal协议处理器: %w", err)
	}
	raw, err := handler.Codec().Encode(packet, protocol.AuthContext{SharedSecret: []byte(secret)})
	if err != nil {
		return fmt.Errorf("编码Portal报文: %w", err)
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(c.nasPort(port))))
	if err != nil {
		return fmt.Errorf("解析Portal NAS地址: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("连接Portal NAS: %w", err)
	}
	defer conn.Close()
	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("设置Portal写超时: %w", err)
	}
	if _, err := conn.Write(raw); err != nil {
		return fmt.Errorf("发送Portal报文: %w", err)
	}
	return nil
}

// DecodeAndVerify validates an inbound NAS-initiated packet without I/O.
func (c *ProtocolClient) DecodeAndVerify(profile protocol.Profile, raw []byte, secret string) (*protocol.Packet, error) {
	handler, err := c.handler(profile)
	if err != nil {
		return nil, fmt.Errorf("解析Portal入站报文: %w", err)
	}
	packet, err := handler.Codec().Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("校验Portal入站报文: %w", err)
	}
	if err := handler.Codec().Verify(packet, protocol.AuthContext{SharedSecret: []byte(secret)}); err != nil {
		return nil, fmt.Errorf("校验Portal入站报文: %w", err)
	}
	return packet, nil
}

func (c *ProtocolClient) handler(profile protocol.Profile) (protocol.ProtocolHandler, error) {
	if c == nil || c.registry == nil {
		return nil, fmt.Errorf("portal: 协议运行时未启用")
	}
	return c.registry.Handler(profile)
}
func (c *ProtocolClient) nasPort(port int) int {
	if port > 0 {
		return port
	}
	if c.port > 0 {
		return c.port
	}
	return NASPort
}

var errorsMismatchedResponse = fmt.Errorf("portal: 响应与请求不匹配")
