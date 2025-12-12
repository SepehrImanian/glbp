package netio

import (
	"fmt"
	"net"

	"github.com/SepehrImanian/glbp/internal/config"
	"github.com/SepehrImanian/glbp/internal/domain"
	"github.com/SepehrImanian/glbp/internal/ports"
)

type LocalInfo struct {
	cfg *config.Config
}

var _ ports.LocalInfo = (*LocalInfo)(nil)

func NewLocalInfo(cfg *config.Config) (*LocalInfo, error) {
	return &LocalInfo{cfg: cfg}, nil
}

func (l *LocalInfo) LocalIP() (net.IP, error) {
	ifi, err := net.InterfaceByName(l.cfg.Interface)
	if err != nil {
		return nil, err
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4, nil
		}
	}
	return nil, fmt.Errorf("no IPv4 found on %s", l.cfg.Interface)
}

func (l *LocalInfo) LocalForwarders() ([]domain.Forwarder, error) {
	out := make([]domain.Forwarder, 0, len(l.cfg.Forwarders))
	for _, f := range l.cfg.Forwarders {
		ifi, err := net.InterfaceByName(f.Iface)
		if err != nil {
			return nil, err
		}
		if len(ifi.HardwareAddr) != 6 {
			return nil, fmt.Errorf("iface %s invalid MAC", f.Iface)
		}
		mac := make(net.HardwareAddr, 6)
		copy(mac, ifi.HardwareAddr)
		out = append(out, domain.Forwarder{
			ID:      f.ID,
			OwnerIP: nil, // filled by app
			MAC:     mac,
			Weight:  f.Weight,
		})
	}
	return out, nil
}
