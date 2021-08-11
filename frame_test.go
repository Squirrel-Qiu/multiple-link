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
	f := Frame{
		ver:    1,
		cmd:    0,
		length: uint16(len(s)),
		fid:    3,
		data:   s,
	}

	tests := []struct {
		name    string
		args    args
		want    Frame
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{newBuffer},
			want:    f,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalFrame(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalFrame() got = %v, want %v", got, tt.want)
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

	f := Frame{
		ver:    1,
		cmd:    0,
		length: uint16(len(s)),
		fid:    3,
		data:   s,
	}

	type args struct {
		f Frame
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
			got := MarshalFrame(tt.arg.f)
			//if (err != nil) != tt.wantErr {
			//	t.Errorf("MarshalFrame() error = %v, wantErr %v", err, tt.wantErr)
			//	return
			//}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalFrame() got = %v, want %v", got, tt.want)
			}
		})
	}
}
