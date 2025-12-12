package ports

import (
	"context"
	"net"
	"time"
)

type ForwarderTLV struct {
	ID     uint8
	MAC    [6]byte
	Weight uint8
}

type HelloMessage struct {
	GroupID   uint16
	VirtualIP net.IP
	Priority  uint8
	Weight    uint8
	Role      uint8 // mapped in app layer

	Forwarders []ForwarderTLV
}

type HelloBus interface {
	Run(ctx context.Context, onHello func(src net.IP, at time.Time, msg HelloMessage)) error
	Send(ctx context.Context, msg HelloMessage) error
}
