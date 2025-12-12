package ports

import (
	"time"

	"github.com/SepehrImanian/glbp/internal/domain"
)

type Repo interface {
	UpsertPeer(domain.Peer)
	ListPeers(now time.Time, hold time.Duration) []domain.Peer

	UpsertForwarder(domain.Forwarder)
	ListForwarders(now time.Time, hold time.Duration) []domain.Forwarder

	RemoveStale(now time.Time, hold time.Duration)
}
