package io

import (
    "context"
    "fmt"
    "log"
    "net"
    "time"

    "example.com/glbpd/internal/config"
    "example.com/glbpd/internal/core"
    "example.com/glbpd/internal/proto"
)

type HelloIO struct {
    iface *net.Interface
    conn  *net.UDPConn
    addr  *net.UDPAddr

    group *core.Group

    helloInterval time.Duration
}

func NewHelloIO(ifaceName, mcastGroup string, port int, g *core.Group) (*HelloIO, error) {
    iface, err := net.InterfaceByName(ifaceName)
    if err != nil {
        return nil, err
    }

    addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", mcastGroup, port))
    if err != nil {
        return nil, err
    }

    conn, err := net.ListenMulticastUDP("udp4", iface, addr)
    if err != nil {
        return nil, err
    }
    if err := conn.SetReadBuffer(65535); err != nil {
        return nil, err
    }

    return &HelloIO{
        iface: iface,
        conn:  conn,
        addr:  addr,
        group: g,
        helloInterval: 3 * time.Second,
    }, nil
}

func (h *HelloIO) SetFromConfig(cfg *config.Config) {
    h.helloInterval = time.Duration(cfg.HelloTimeSec) * time.Second
}

func (h *HelloIO) Run(ctx context.Context) error {
    go h.rxLoop(ctx)
    return h.txLoop(ctx)
}

func (h *HelloIO) rxLoop(ctx context.Context) {
    buf := make([]byte, 1500)
    for {
        h.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
        n, src, err := h.conn.ReadFromUDP(buf)
        select {
        case <-ctx.Done():
            return
        default:
        }
        if err != nil {
            if ne, ok := err.(net.Error); ok && ne.Timeout() {
                continue
            }
            log.Printf("hello rx error: %v", err)
            continue
        }
        data := buf[:n]
        msg, err := proto.DecodeHello(data)
        if err != nil {
            continue
        }
        h.group.OnHello(msg, src.IP, time.Now())
    }
}

func (h *HelloIO) txLoop(ctx context.Context) error {
    ticker := time.NewTicker(h.helloInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            if err := h.sendHello(); err != nil {
                log.Printf("hello tx error: %v", err)
            }
        }
    }
}

func (h *HelloIO) sendHello() error {
    vgState := proto.VGListen
    if h.group.IsActiveVG() {
        vgState = proto.VGActive
    } else if h.group.GatewayState == core.GWStandby {
        vgState = proto.VGStandby
    }

    msg := &proto.HelloMessage{
        Header: proto.GLBPHeader{
            Version:   1,
            OpCode:    proto.OpHello,
            Group:     h.group.ID,
            Priority:  h.group.LocalPrio,
            Weight:    h.group.LocalWeight,
            VGState:   vgState,
            VirtualIP: h.group.VirtualIP,
        },
        Forwarders: nil,
    }

    payload, err := proto.EncodeHello(msg)
    if err != nil {
        return err
    }
    _, err = h.conn.WriteToUDP(payload, h.addr)
    return err
}
