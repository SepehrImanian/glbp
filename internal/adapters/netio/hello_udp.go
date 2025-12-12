package netio

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/SepehrImanian/glbp/internal/config"
	"github.com/SepehrImanian/glbp/internal/ports"
)

type HelloUDP struct {
	cfg  *config.Config
	conn *net.UDPConn
	mcast *net.UDPAddr
	iface *net.Interface
}

func NewHelloUDP(cfg *config.Config) (*HelloUDP, error) {
	iface, err := net.InterfaceByName(cfg.Interface)
	if err != nil {
		return nil, err
	}
	maddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", cfg.MulticastGroup, cfg.MulticastPort))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenMulticastUDP("udp4", iface, maddr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetReadBuffer(1 << 20)

	return &HelloUDP{cfg: cfg, conn: conn, mcast: maddr, iface: iface}, nil
}

func (h *HelloUDP) Run(ctx context.Context, cb func(net.IP, time.Time, ports.HelloMessage)) error {
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_ = h.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, src, err := h.conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		msg, err := decodeHello(buf[:n])
		if err != nil {
			continue
		}
		cb(src.IP, time.Now(), msg)
	}
}

func (h *HelloUDP) Send(ctx context.Context, msg ports.HelloMessage) error {
	b, err := encodeHello(msg)
	if err != nil {
		return err
	}
	_, err = h.conn.WriteToUDP(b, h.mcast)
	return err
}
