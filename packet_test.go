package multiple_link

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"testing"
)

func TestUnmarshalFrame(t *testing.T) {
	s := []byte("ping")
	data := make([]byte, 8+len(s))
	data[0] = 1
	data[1] = 0
	binary.BigEndian.PutUint32(data[2:6], 3)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(s)))
	copy(data[8:], s)

	newBuffer := bytes.NewBuffer(data)

	type args struct {
		r io.Reader
	}
	p := Packet{
		ver:    1,
		cmd:    0,
		length: uint16(len(s)),
		pid:    3,
		data:   s,
	}

	tests := []struct {
		name    string
		args    args
		want    Packet
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{newBuffer},
			want:    p,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalPacket(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("UnmarshalPacket() got = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestMarshalFrame(t *testing.T) {
	s := []byte("ping")
	data := make([]byte, 8+len(s))
	data[0] = 1
	data[1] = 0
	binary.BigEndian.PutUint32(data[2:6], 3)
	binary.BigEndian.PutUint16(data[6:8], uint16(len(s)))
	copy(data[8:], s)

	p := &Packet{
		ver:    1,
		cmd:    0,
		length: uint16(len(s)),
		pid:    3,
		data:   s,
	}

	type args struct {
		p *Packet
	}
	tests := []struct {
		name    string
		arg     args
		want    []byte
		wantErr bool
	}{
		{
			name:    "success2",
			arg:     args{p},
			want:    data,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarshalPacket(tt.arg.p)
			//if (err != nil) != tt.wantErr {
			//	t.Errorf("MarshalPacket() error = %v, wantErr %v", err, tt.wantErr)
			//	return
			//}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalPacket() got = %v, want %v", got, tt.want)
			}
		})
	}
}
