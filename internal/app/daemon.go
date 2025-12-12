package app

import (
	"context"
	"net"
	"sort"
	"time"

	"github.com/SepehrImanian/glbp/internal/domain"
	"github.com/SepehrImanian/glbp/internal/ports"
)

type Logger interface {
	Printf(format string, v ...any)
}

type Daemon struct {
	GroupID   uint16
	VirtualIP net.IP

	LocalPrio   uint8
	LocalWeight uint8
	Preempt     bool

	HelloInterval time.Duration
	HoldTime      time.Duration

	Hello ports.HelloBus
	ARP   ports.ARPResponder
	Repo  ports.Repo
	Info  ports.LocalInfo

	Selector domain.ForwarderSelector
	Logger   Logger
}

func (d *Daemon) Run(ctx context.Context) error {
	localIP, err := d.Info.LocalIP()
	if err != nil {
		return err
	}
	localIP = localIP.To4()

	if d.Logger != nil {
		d.Logger.Printf("starting: if VIP=%s group=%d prio=%d hello=%s hold=%s selector=%s",
			d.VirtualIP, d.GroupID, d.LocalPrio, d.HelloInterval, d.HoldTime, d.Selector.Name())
	}

	// Hello receiver
	go func() {
		_ = d.Hello.Run(ctx, func(src net.IP, at time.Time, msg ports.HelloMessage) {
			if msg.GroupID != d.GroupID {
				return
			}
			d.Repo.UpsertPeer(domain.Peer{
				IP:       src.To4(),
				Priority: msg.Priority,
				Weight:   msg.Weight,
				Role:     domain.GatewayRole(msg.Role),
				LastSeen: at,
			})
			for _, f := range msg.Forwarders {
				mac := make(net.HardwareAddr, 6)
				copy(mac, f.MAC[:])
				d.Repo.UpsertForwarder(domain.Forwarder{
					ID:       f.ID,
					OwnerIP:  src.To4(),
					MAC:      mac,
					Weight:   f.Weight,
					LastSeen: at,
				})
			}
		})
	}()

	// ARP listener
	go func() {
		_ = d.ARP.Run(ctx, func(req ports.ARPRequest) {
			if !req.DstIP.Equal(d.VirtualIP) {
				return
			}
			role := d.currentRole(localIP)
			if role != domain.RoleAVG {
				return
			}
			fwds := d.Repo.ListForwarders(time.Now(), d.HoldTime)
			sort.Slice(fwds, func(i, j int) bool { return fwds[i].ID < fwds[j].ID })
			sel := d.Selector.Select(fwds)
			if sel == nil {
				return
			}
			_ = d.ARP.Reply(ctx, req, d.VirtualIP, sel.MAC)
		})
	}()

	ticker := time.NewTicker(d.HelloInterval)
	defer ticker.Stop()

	// initial populate local forwarders into repo
	if err := d.publishLocalForwarders(localIP); err != nil && d.Logger != nil {
		d.Logger.Printf("warning: local forwarders: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			if d.Logger != nil {
				d.Logger.Printf("stopped")
			}
			return nil
		case now := <-ticker.C:
			d.Repo.RemoveStale(now, d.HoldTime)
			_ = d.publishLocalForwarders(localIP) // refresh timestamps + repo
			role := d.currentRole(localIP)
			_ = d.Hello.Send(ctx, ports.HelloMessage{
				GroupID:    d.GroupID,
				VirtualIP:  d.VirtualIP,
				Priority:   d.LocalPrio,
				Weight:     d.LocalWeight,
				Role:       uint8(role),
				Forwarders: d.localForwarderTLVs(localIP),
			})
			if d.Logger != nil {
				d.Logger.Printf("tick role=%v peers=%d fwds=%d", role, len(d.Repo.ListPeers(now, d.HoldTime)), len(d.Repo.ListForwarders(now, d.HoldTime)))
			}
		}
	}
}

func (d *Daemon) publishLocalForwarders(localIP net.IP) error {
	fwds, err := d.Info.LocalForwarders()
	if err != nil {
		return err
	}
	now := time.Now()
	for _, f := range fwds {
		f.OwnerIP = localIP
		f.LastSeen = now
		d.Repo.UpsertForwarder(f)
	}
	return nil
}

func (d *Daemon) localForwarderTLVs(localIP net.IP) []ports.ForwarderTLV {
	fwds, err := d.Info.LocalForwarders()
	if err != nil {
		return nil
	}
	out := make([]ports.ForwarderTLV, 0, len(fwds))
	for _, f := range fwds {
		var mac [6]byte
		copy(mac[:], f.MAC)
		out = append(out, ports.ForwarderTLV{ID: f.ID, Weight: f.Weight, MAC: mac})
	}
	return out
}

func (d *Daemon) currentRole(localIP net.IP) domain.GatewayRole {
	now := time.Now()
	peers := d.Repo.ListPeers(now, d.HoldTime)

	bestPrio := d.LocalPrio
	bestIP := localIP.To4()
	bestIsLocal := true

	var activePeer *domain.Peer
	for i := range peers {
		p := peers[i]
		if p.Role == domain.RoleAVG {
			activePeer = &p
		}
		if p.Priority > bestPrio || (p.Priority == bestPrio && ipGreater(p.IP, bestIP)) {
			bestPrio = p.Priority
			bestIP = p.IP.To4()
			bestIsLocal = false
		}
	}

	if activePeer == nil {
		if bestIsLocal {
			return domain.RoleAVG
		}
		return domain.RoleStandby
	}

	if d.Preempt && d.LocalPrio > activePeer.Priority {
		return domain.RoleAVG
	}
	if bestIsLocal {
		return domain.RoleStandby
	}
	return domain.RoleListen
}

func ipGreater(a, b net.IP) bool {
	aa, bb := a.To4(), b.To4()
	if aa == nil || bb == nil {
		return false
	}
	for i := 0; i < 4; i++ {
		if aa[i] > bb[i] {
			return true
		}
		if aa[i] < bb[i] {
			return false
		}
	}
	return false
}
