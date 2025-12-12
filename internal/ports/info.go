package ports

import (
	"net"

	"github.com/SepehrImanian/glbp/internal/domain"
)

type LocalInfo interface {
	LocalIP() (net.IP, error)
	LocalForwarders() ([]domain.Forwarder, error)
}
