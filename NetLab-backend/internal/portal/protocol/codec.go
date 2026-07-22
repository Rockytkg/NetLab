package protocol

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
)

type Profile string

const (
	ProfileCMCCV1   Profile = "cmcc-v1"
	ProfileCMCCV2   Profile = "cmcc-v2"
	ProfileHuaweiV1 Profile = "huawei-v1"
	ProfileHuaweiV2 Profile = "huawei-v2"
)

type AuthContext struct {
	SharedSecret         []byte
	RequestAuthenticator *[16]byte
}
type Codec interface {
	Decode(raw []byte) (*Packet, error)
	Encode(packet *Packet, auth AuthContext) ([]byte, error)
	Verify(packet *Packet, auth AuthContext) error
}
type ProtocolHandler interface {
	Profile() Profile
	Codec() Codec
	Validate(packet *Packet) error
	Handle(ctx context.Context, packet *Packet, peer netip.Addr) (*Packet, error)
}

// Registry is deliberately explicit: the composition root registers protocol
// implementations instead of relying on package init side effects.
type Registry struct {
	mu       sync.RWMutex
	handlers map[Profile]ProtocolHandler
}

func NewRegistry() *Registry { return &Registry{handlers: make(map[Profile]ProtocolHandler)} }
func (r *Registry) Register(handler ProtocolHandler) error {
	if handler == nil {
		return fmt.Errorf("%w: nil", ErrHandlerNotFound)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile := handler.Profile()
	if _, ok := r.handlers[profile]; ok {
		return fmt.Errorf("%w: %s", ErrHandlerAlreadyExists, profile)
	}
	r.handlers[profile] = handler
	return nil
}
func (r *Registry) Handler(profile Profile) (ProtocolHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[profile]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrHandlerNotFound, profile)
	}
	return handler, nil
}
