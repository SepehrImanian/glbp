package core

import (
    "net"
    "sync"
    "time"

    "example.com/glbpd/internal/config"
    "example.com/glbpd/internal/proto"
)

type GatewayState int

const (
    GWInit GatewayState = iota
    GWListen
    GWStandby
    GWActive
)

type GatewayTimers struct {
    HelloTime time.Duration
    HoldTime  time.Duration
}

type Peer struct {
    IP       net.IP
    Priority uint8
    Weight   uint8
    State    proto.VGState
    LastSeen time.Time
}

type Forwarder struct {
    ID         uint8
    State      uint8
    VirtualMAC net.HardwareAddr
    Weight     uint8
}

type Group struct {
    mu sync.RWMutex

    ID        uint16
    VirtualIP net.IP

    LocalIP      net.IP
    LocalPrio    uint8
    LocalWeight  uint8
    Preempt      bool
    GatewayState GatewayState
    Timers       GatewayTimers

    Peers map[string]*Peer
    Forwarders map[uint8]*Forwarder

    rrIndex int
}

func NewGroup(cfg *config.Config) (*Group, error) {
    return &Group{
        ID:         uint16(cfg.GroupID),
        VirtualIP:  net.ParseIP(cfg.VirtualIP).To4(),
        LocalPrio:  cfg.Priority,
        LocalWeight: cfg.Weight,
        Preempt:    cfg.Preempt,
        GatewayState: GWInit,
        Timers: GatewayTimers{
            HelloTime: time.Duration(cfg.HelloTimeSec) * time.Second,
            HoldTime:  time.Duration(cfg.HoldTimeSec) * time.Second,
        },
        Peers:      make(map[string]*Peer),
        Forwarders: make(map[uint8]*Forwarder),
    }, nil
}

func (g *Group) OnHello(msg *proto.HelloMessage, src net.IP, now time.Time) {
    g.mu.Lock()
    defer g.mu.Unlock()

    if msg.Header.Group != g.ID {
        return
    }

    key := src.String()
    p, ok := g.Peers[key]
    if !ok {
        p = &Peer{IP: src}
        g.Peers[key] = p
    }
    p.Priority = msg.Header.Priority
    p.Weight = msg.Header.Weight
    p.State = msg.Header.VGState
    p.LastSeen = now

    for _, ft := range msg.Forwarders {
        mac := make(net.HardwareAddr, 6)
        copy(mac, ft.VirtualMAC[:])
        g.Forwarders[ft.ForwarderID] = &Forwarder{
            ID:         ft.ForwarderID,
            State:      ft.State,
            VirtualMAC: mac,
            Weight:     ft.Weight,
        }
    }

    g.recomputeGatewayState(now)
}

func (g *Group) OnTimerTick(now time.Time) {
    g.mu.Lock()
    defer g.mu.Unlock()

    for k, p := range g.Peers {
        if now.Sub(p.LastSeen) > g.Timers.HoldTime {
            delete(g.Peers, k)
        }
    }
    g.recomputeGatewayState(now)
}

func (g *Group) recomputeGatewayState(now time.Time) {
    highestPrio := g.LocalPrio
    highestIsLocal := true
    var activePeer *Peer

    for _, p := range g.Peers {
        if p.State == proto.VGActive {
            activePeer = p
        }
        if p.Priority > highestPrio || (p.Priority == highestPrio && ipGreater(p.IP, g.LocalIP)) {
            highestPrio = p.Priority
            highestIsLocal = false
        }
    }

    if activePeer == nil {
        if highestIsLocal {
            g.GatewayState = GWActive
        } else {
            g.GatewayState = GWStandby
        }
        return
    }

    if g.Preempt && g.LocalPrio > activePeer.Priority {
        g.GatewayState = GWActive
        return
    }

    if highestIsLocal {
        g.GatewayState = GWStandby
    } else {
        g.GatewayState = GWListen
    }
}

func ipGreater(a, b net.IP) bool {
    if a == nil || b == nil {
        return false
    }
    aa := a.To4()
    bb := b.To4()
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

func (g *Group) IsActiveVG() bool {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g.GatewayState == GWActive
}

func (g *Group) SelectForwarderMAC(hostIP net.IP) net.HardwareAddr {
    g.mu.RLock()
    defer g.mu.RUnlock()

    if len(g.Forwarders) == 0 {
        return nil
    }

    ids := make([]uint8, 0, len(g.Forwarders))
    for id := range g.Forwarders {
        ids = append(ids, id)
    }
    for i := 0; i < len(ids); i++ {
        for j := i + 1; j < len(ids); j++ {
            if ids[j] < ids[i] {
                ids[i], ids[j] = ids[j], ids[i]
            }
        }
    }
    idx := g.rrIndex % len(ids)
    g.rrIndex++
    fwd := g.Forwarders[ids[idx]]
    return fwd.VirtualMAC
}
