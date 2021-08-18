package multiple_link

import (
	"encoding/binary"
	"io"

	"golang.org/x/xerrors"
)

const VERSION = 1

const (
	cmdSYN byte = iota
	cmdPSH
	cmdPing
	cmdClose
)

type Packet struct {
	ver    byte
	cmd    byte
	pid    uint32
	length uint16
	data   []byte
}

func newPacket(version, cmd byte, pid uint32) Packet {
	return Packet{ver: version, cmd: cmd, pid: pid}
}

func UnmarshalPacket(r io.Reader) (*Packet, error) {
	b := make([]byte, 8)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, xerrors.Errorf("unmarshal packet from net.Conn failed: %w", err)
	}

	p := new(Packet)

	p.ver = b[0]
	if p.ver != VERSION {
		return nil, xerrors.Errorf("unmarshal packet's version from net.Conn isn't %w", p.ver)
	}

	switch b[1] {
	case cmdSYN, cmdPSH, cmdPing, cmdClose:
		p.cmd = b[1]
	default:
		return nil, xerrors.Errorf("unmarshal packet's cmd from net.Conn isn't %w", b[1])
	}

	p.pid = binary.BigEndian.Uint32(b[2:6])

	p.length = binary.BigEndian.Uint16(b[6:8])
	if p.length != 0 {
		p.data = make([]byte, p.length)
		if _, err := io.ReadFull(r, p.data); err != nil {
			return nil, xerrors.Errorf("unmarshal packet's data from net.Conn failed: %w", err)
		}
	}

	return p, nil
}

func MarshalPacket(p Packet) []byte {
	buf := make([]byte, 8+p.length)
	buf[0] = p.ver
	buf[1] = p.cmd
	binary.BigEndian.PutUint16(buf[2:4], p.length)
	binary.BigEndian.PutUint32(buf[4:8], p.pid)
	copy(buf[8:], p.data)
	return buf
}