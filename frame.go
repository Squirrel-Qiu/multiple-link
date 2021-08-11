package multiple_link

import (
	"encoding/binary"
	"errors"
	"io"
)

const VERSION = 1

const (
	cmdSYN byte = iota
	cmdPSH
	cmdPing
	cmdClose
)

type Frame struct {
	ver    byte
	cmd    byte
	length uint16
	fid    uint32
	data   []byte
}

func UnmarshalFrame(r io.Reader) (Frame, error) {
	verBytes := make([]byte, 1)

	_, err := r.Read(verBytes)
	if err != nil {
		return Frame{}, err
	}

	if verBytes[0] != VERSION {
		return Frame{}, err
	}

	cmdBytes := make([]byte, 1)

	_, err = r.Read(cmdBytes)
	if err != nil {
		return Frame{}, err
	}

	switch cmdBytes[0] {
	case cmdSYN:
		// newLink
	case cmdPSH:
	case cmdClose:
	default:
		return Frame{}, errors.New("cmd is false")
	}

	frameLen := make([]byte, 2)

	if _, err := io.ReadFull(r, frameLen); err != nil {
		return Frame{}, err
	}

	if binary.BigEndian.Uint16(frameLen) == 0 {
		return Frame{}, errors.New("data length is zero")
	}

	sidBytes := make([]byte, 4)
	if _, err := io.ReadFull(r, sidBytes); err != nil {
		return Frame{}, err
	}

	f := Frame{
		ver:    verBytes[0],
		cmd:    cmdBytes[0],
		length: binary.BigEndian.Uint16(frameLen),
		fid:    binary.BigEndian.Uint32(sidBytes),
	}

	dataBytes := make([]byte, f.length)
	if _, err := io.ReadFull(r, dataBytes); err != nil {
		return Frame{}, err
	}

	f.data = dataBytes

	return f, nil
}

func MarshalFrame(f Frame) []byte {
	buf := make([]byte, 8+f.length)
	buf[0] = f.ver
	buf[1] = f.cmd
	binary.BigEndian.PutUint16(buf[2:4], f.length)
	binary.BigEndian.PutUint32(buf[4:8], f.fid)
	copy(buf[8:], f.data)
	return buf
}
