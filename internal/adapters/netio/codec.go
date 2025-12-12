package netio

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/SepehrImanian/glbp/internal/ports"
)

// This is a simple lab codec (NOT Cisco GLBP wire format).
// Layout:
// [0]=ver(1)
// [1]=role
// [2..3]=group
// [4]=prio
// [5]=weight
// [6..9]=vip IPv4
// [10]=fwdCount
// then each forwarder: [id][weight][mac(6)] => 8 bytes
const (
	codecVersion = 1
)

func encodeHello(m ports.HelloMessage) ([]byte, error) {
	vip := m.VirtualIP.To4()
	if vip == nil {
		return nil, fmt.Errorf("vip must be IPv4")
	}
	fc := len(m.Forwarders)
	buf := make([]byte, 11+8*fc)
	buf[0] = codecVersion
	buf[1] = m.Role
	binary.BigEndian.PutUint16(buf[2:4], m.GroupID)
	buf[4] = m.Priority
	buf[5] = m.Weight
	copy(buf[6:10], vip)
	buf[10] = byte(fc)
	off := 11
	for _, f := range m.Forwarders {
		buf[off] = f.ID
		buf[off+1] = f.Weight
		copy(buf[off+2:off+8], f.MAC[:])
		off += 8
	}
	return buf, nil
}

func decodeHello(b []byte) (ports.HelloMessage, error) {
	if len(b) < 11 {
		return ports.HelloMessage{}, fmt.Errorf("too short")
	}
	if b[0] != codecVersion {
		return ports.HelloMessage{}, fmt.Errorf("bad version")
	}
	m := ports.HelloMessage{
		Role:     b[1],
		GroupID:  binary.BigEndian.Uint16(b[2:4]),
		Priority: b[4],
		Weight:   b[5],
		VirtualIP: net.IPv4(b[6], b[7], b[8], b[9]).To4(),
	}
	fc := int(b[10])
	exp := 11 + 8*fc
	if len(b) < exp {
		return ports.HelloMessage{}, fmt.Errorf("bad length")
	}
	off := 11
	m.Forwarders = make([]ports.ForwarderTLV, 0, fc)
	for i := 0; i < fc; i++ {
		var mac [6]byte
		copy(mac[:], b[off+2:off+8])
		m.Forwarders = append(m.Forwarders, ports.ForwarderTLV{
			ID:     b[off],
			Weight: b[off+1],
			MAC:    mac,
		})
		off += 8
	}
	return m, nil
}
