package domain

import (
	"net"
	"time"
)

type Peer struct {
	IP       net.IP
	Priority uint8
	Weight   uint8
	Role     GatewayRole
	LastSeen time.Time
}

type Forwarder struct {
	ID       uint8
	OwnerIP  net.IP
	MAC      net.HardwareAddr
	Weight   uint8
	LastSeen time.Time
}
