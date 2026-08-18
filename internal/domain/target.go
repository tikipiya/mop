package domain

import (
	"net"
	"strconv"
)

const DefaultPort uint16 = 25565

// Target is the normalized destination used for the Minecraft handshake.
// UseSRV records that the user omitted a port and permits DNS discovery for
// public hostnames before the TCP connection is opened.
type Target struct {
	Host   string
	Port   uint16
	UseSRV bool
}

func (t Target) Address() string {
	return net.JoinHostPort(t.Host, strconv.FormatUint(uint64(t.Port), 10))
}
