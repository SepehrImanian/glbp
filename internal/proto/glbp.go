package proto

import (
    "encoding/binary"
    "fmt"
    "net"
)

type OpCode uint8

const (
    OpHello OpCode = 1
)

type VGState uint8

const (
    VGListen VGState = 0
    VGActive VGState = 1
    VGStandby VGState = 2
)

// GLBPHeader is a simplified and NOT protocol-accurate header.
// Replace with real GLBP layout for interoperability.
type GLBPHeader struct {
    Version uint8
    OpCode  OpCode
    Group   uint16

    Priority uint8
    Weight   uint8

    VGState VGState

    VirtualIP net.IP // 4 bytes
}

type ForwarderTLV struct {
    ForwarderID uint8
    State       uint8
    VirtualMAC  [6]byte
    Weight      uint8
}

type HelloMessage struct {
    Header     GLBPHeader
    Forwarders []ForwarderTLV
}

func EncodeHello(msg *HelloMessage) ([]byte, error) {
    if msg.Header.VirtualIP.To4() == nil {
        return nil, fmt.Errorf("only IPv4 supported in this skeleton")
    }
    fwdCount := len(msg.Forwarders)
    buf := make([]byte, 12+9*fwdCount)

    buf[0] = msg.Header.Version
    buf[1] = uint8(msg.Header.OpCode)
    binary.BigEndian.PutUint16(buf[2:], msg.Header.Group)
    buf[4] = msg.Header.Priority
    buf[5] = msg.Header.Weight
    buf[6] = uint8(msg.Header.VGState)
    copy(buf[7:11], msg.Header.VirtualIP.To4())
    buf[11] = byte(fwdCount)

    off := 12
    for _, f := range msg.Forwarders {
        buf[off] = f.ForwarderID
        buf[off+1] = f.State
        copy(buf[off+2:off+8], f.VirtualMAC[:])
        buf[off+8] = f.Weight
        off += 9
    }
    return buf, nil
}

func DecodeHello(b []byte) (*HelloMessage, error) {
    if len(b) < 12 {
        return nil, fmt.Errorf("hello too short")
    }
    h := GLBPHeader{
        Version:  b[0],
        OpCode:   OpCode(b[1]),
        Group:    binary.BigEndian.Uint16(b[2:4]),
        Priority: b[4],
        Weight:   b[5],
        VGState:  VGState(b[6]),
        VirtualIP: net.IPv4(b[7], b[8], b[9], b[10]),
    }
    fwdCount := int(b[11])
    expectedLen := 12 + 9*fwdCount
    if len(b) < expectedLen {
        return nil, fmt.Errorf("invalid forwarder count/length")
    }
    fwd := make([]ForwarderTLV, 0, fwdCount)
    off := 12
    for i := 0; i < fwdCount; i++ {
        var mac [6]byte
        copy(mac[:], b[off+2:off+8])
        fwd = append(fwd, ForwarderTLV{
            ForwarderID: b[off],
            State:       b[off+1],
            VirtualMAC:  mac,
            Weight:      b[off+8],
        })
        off += 9
    }
    return &HelloMessage{
        Header:     h,
        Forwarders: fwd,
    }, nil
}
