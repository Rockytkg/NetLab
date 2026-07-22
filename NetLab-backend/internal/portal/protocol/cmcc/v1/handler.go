package v1

import (
	"context"
	"net/netip"

	"netlab-backend/internal/portal/protocol"
)

type Processor interface {
	Process(context.Context, *protocol.Packet, netip.Addr) (*protocol.Packet, error)
}
type Handler struct {
	processor Processor
	codec     Codec
}

func NewHandler(processor Processor) *Handler {
	return &Handler{processor: processor, codec: NewCodec()}
}
func (h *Handler) Profile() protocol.Profile              { return protocol.ProfileCMCCV1 }
func (h *Handler) Codec() protocol.Codec                  { return h.codec }
func (h *Handler) Validate(packet *protocol.Packet) error { return Validate(packet) }
func (h *Handler) Handle(ctx context.Context, packet *protocol.Packet, peer netip.Addr) (*protocol.Packet, error) {
	if err := h.Validate(packet); err != nil {
		return nil, err
	}
	if h.processor == nil {
		return nil, nil
	}
	return h.processor.Process(ctx, packet, peer)
}
