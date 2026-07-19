package handlers

import (
	"crypto/rand"
	"fmt"
	"time"
)

// uuid generates a time-based UUID string with cryptographic randomness, used
// for RADIUS State attribute values. The format is:
// {unix32bits}-{rand}-{rand}-{rand}-{rand}-{rand}
// (Ported from toughradius pkg/common.UUID; kept local because
// netlab-backend has no pkg/common package.)
func uuid() string {
	unix32bits := uint32(time.Now().UTC().Unix()) //nolint:gosec // G115: Unix timestamp fits in uint32 until 2106
	buff := make([]byte, 12)
	numRead, err := rand.Read(buff)
	if numRead != len(buff) || err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x-%x", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
}
