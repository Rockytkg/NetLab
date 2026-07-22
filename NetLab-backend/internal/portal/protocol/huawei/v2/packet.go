package v2

import "netlab-backend/internal/portal/protocol"

const (
	Version      = protocol.VersionV2
	HeaderLength = protocol.HeaderV2Length
)

type Packet = protocol.Packet
type Attribute = protocol.Attribute
