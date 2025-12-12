package repo

import (
	"sync"
	"time"

	"github.com/SepehrImanian/glbp/internal/domain"
)

type Memory struct {
	mu    sync.Mutex
	peers map[string]domain.Peer
	fwds  map[uint8]domain.Forwarder
}

func NewMemory() *Memory {
	return &Memory{
		peers: make(map[string]domain.Peer),
		fwds:  make(map[uint8]domain.Forwarder),
	}
}

func (m *Memory) UpsertPeer(p domain.Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[p.IP.String()] = p
}

func (m *Memory) ListPeers(now time.Time, hold time.Duration) []domain.Peer {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Peer, 0, len(m.peers))
	for _, p := range m.peers {
		if now.Sub(p.LastSeen) <= hold {
			out = append(out, p)
		}
	}
	return out
}

func (m *Memory) UpsertForwarder(f domain.Forwarder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fwds[f.ID] = f
}

func (m *Memory) ListForwarders(now time.Time, hold time.Duration) []domain.Forwarder {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Forwarder, 0, len(m.fwds))
	for _, f := range m.fwds {
		if now.Sub(f.LastSeen) <= hold {
			out = append(out, f)
		}
	}
	return out
}

func (m *Memory) RemoveStale(now time.Time, hold time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, p := range m.peers {
		if now.Sub(p.LastSeen) > hold {
			delete(m.peers, k)
		}
	}
	for id, f := range m.fwds {
		if now.Sub(f.LastSeen) > hold {
			delete(m.fwds, id)
		}
	}
}
