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
	binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
	binary.BigEndian.PutUint32(data[4:8], 3)
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalPacket() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalFrame(t *testing.T) {
	s := []byte("ping")
	data := make([]byte, 8+len(s))
	data[0] = 1
	data[1] = 0
	binary.BigEndian.PutUint16(data[2:4], uint16(len(s)))
	binary.BigEndian.PutUint32(data[4:8], 3)
	copy(data[8:], s)

	f := Packet{
		ver:    1,
		cmd:    0,
		length: uint16(len(s)),
		pid:    3,
		data:   s,
	}

	type args struct {
		f Packet
	}
	tests := []struct {
		name    string
		arg    args
		want   []byte
		wantErr bool
	}{
		{
			name:    "success2",
			arg:    args{f},
			want:    data,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarshalPacket(tt.arg.f)
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