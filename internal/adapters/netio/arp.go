package netio

import (
	"context"
	"fmt"
	"net"

	"github.com/SepehrImanian/glbp/internal/config"
	"github.com/SepehrImanian/glbp/internal/ports"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type ARP struct {
	cfg *config.Config
	h   *pcap.Handle
}

var _ ports.ARPResponder = (*ARP)(nil)

func NewARP(cfg *config.Config) (*ARP, error) {
	h, err := pcap.OpenLive(cfg.Interface, 65535, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	if err := h.SetBPFFilter("arp"); err != nil {
		h.Close()
		return nil, err
	}
	return &ARP{cfg: cfg, h: h}, nil
}

func (a *ARP) Run(ctx context.Context, onReq func(req ports.ARPRequest)) error {
	src := gopacket.NewPacketSource(a.h, a.h.LinkType())
	packets := src.Packets()
	for {
		select {
		case <-ctx.Done():
			return nil
		case pkt, ok := <-packets:
			if !ok {
				return nil
			}
			arpLayer := pkt.Layer(layers.LayerTypeARP)
			ethLayer := pkt.Layer(layers.LayerTypeEthernet)
			if arpLayer == nil || ethLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			eth := ethLayer.(*layers.Ethernet)
			if arp.Operation != layers.ARPRequest {
				continue
			}
			req := ports.ARPRequest{
				SrcIP:  net.IP(arp.SourceProtAddress).To4(),
				SrcMAC: eth.SrcMAC,
				DstIP:  net.IP(arp.DstProtAddress).To4(),
			}
			if req.SrcIP == nil || req.DstIP == nil {
				continue
			}
			onReq(req)
		}
	}
}

func (a *ARP) Reply(ctx context.Context, req ports.ARPRequest, vip net.IP, vmac net.HardwareAddr) error {
	if vip.To4() == nil || len(vmac) != 6 {
		return fmt.Errorf("invalid vip or vmac")
	}
	replyEth := &layers.Ethernet{
		SrcMAC:       vmac,
		DstMAC:       req.SrcMAC,
		EthernetType: layers.EthernetTypeARP,
	}
	replyARP := &layers.ARP{
		Operation:         layers.ARPReply,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		SourceHwAddress:   []byte(vmac),
		SourceProtAddress: []byte(vip.To4()),
		DstHwAddress:      []byte(req.SrcMAC),
		DstProtAddress:    []byte(req.SrcIP.To4()),
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, replyEth, replyARP); err != nil {
		return err
	}
	return a.h.WritePacketData(buf.Bytes())
}
