package v1

import "netlab-backend/internal/portal/protocol"

const (
	Version      = protocol.VersionV1
	HeaderLength = protocol.HeaderV1Length
)

type Packet = protocol.Packet
type Attribute = protocol.Attribute
