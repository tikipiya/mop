package domain

import (
	"net"
	"strconv"
)

const DefaultPort uint16 = 25565

// Target is the normalized destination used for both the handshake and TCP
// connection. Input validation and normalization belong to the app layer.
type Target struct {
	Host string
	Port uint16
}

func (t Target) Address() string {
	return net.JoinHostPort(t.Host, strconv.FormatUint(uint64(t.Port), 10))
}
