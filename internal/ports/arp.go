package ports

import (
	"context"
	"net"
)

type ARPRequest struct {
	SrcIP  net.IP
	SrcMAC net.HardwareAddr
	DstIP  net.IP
}

type ARPResponder interface {
	Run(ctx context.Context, onReq func(req ARPRequest)) error
	Reply(ctx context.Context, req ARPRequest, vip net.IP, vmac net.HardwareAddr) error
}
