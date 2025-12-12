package io

import (
    "context"
    "fmt"
    "log"
    "net"

    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"

    "example.com/glbpd/internal/core"
)

type ARPResponder struct {
    ifaceName string
    vip       net.IP
    group     *core.Group
}

func NewARPResponder(ifaceName, vip string, g *core.Group) (*ARPResponder, error) {
    ip := net.ParseIP(vip).To4()
    if ip == nil {
        return nil, fmt.Errorf("invalid VIP %q", vip)
    }
    return &ARPResponder{
        ifaceName: ifaceName,
        vip:       ip,
        group:     g,
    }, nil
}

func (a *ARPResponder) Run(ctx context.Context) error {
    handle, err := pcap.OpenLive(a.ifaceName, 65535, true, pcap.BlockForever)
    if err != nil {
        return err
    }
    defer handle.Close()

    if err := handle.SetBPFFilter("arp"); err != nil {
        return err
    }

    src := gopacket.NewPacketSource(handle, handle.LinkType())
    inCh := src.Packets()

    for {
        select {
        case <-ctx.Done():
            return nil
        case pkt, ok := <-inCh:
            if !ok {
                return nil
            }
            a.handlePacket(handle, pkt)
        }
    }
}

func (a *ARPResponder) handlePacket(handle *pcap.Handle, packet gopacket.Packet) {
    arpLayer := packet.Layer(layers.LayerTypeARP)
    ethLayer := packet.Layer(layers.LayerTypeEthernet)
    if arpLayer == nil || ethLayer == nil {
        return
    }
    arp := arpLayer.(*layers.ARP)
    eth := ethLayer.(*layers.Ethernet)

    if arp.Operation != layers.ARPRequest {
        return
    }
    targetIP := net.IP(arp.DstProtAddress)
    if !targetIP.Equal(a.vip) {
        return
    }

    if !a.group.IsActiveVG() {
        return
    }

    hostIP := net.IP(arp.SourceProtAddress)
    mac := a.group.SelectForwarderMAC(hostIP)
    if mac == nil {
        return
    }

    replyEth := &layers.Ethernet{
        SrcMAC:       mac,
        DstMAC:       eth.SrcMAC,
        EthernetType: layers.EthernetTypeARP,
    }

    replyARP := &layers.ARP{
        Operation:        layers.ARPReply,
        HwAddressSize:    6,
        ProtAddressSize:  4,
        AddrType:         layers.LinkTypeEthernet,
        Protocol:         layers.EthernetTypeIPv4,
        SourceHwAddress:  []byte(mac),
        SourceProtAddress: []byte(a.vip.To4()),
        DstHwAddress:      arp.SourceHwAddress,
        DstProtAddress:    arp.SourceProtAddress,
    }

    buf := gopacket.NewSerializeBuffer()
    opts := gopacket.SerializeOptions{FixLengths: true}
    if err := gopacket.SerializeLayers(buf, opts, replyEth, replyARP); err != nil {
        log.Printf("serialize ARP reply: %v", err)
        return
    }

    if err := handle.WritePacketData(buf.Bytes()); err != nil {
        log.Printf("send ARP reply: %v", err)
    }
}
