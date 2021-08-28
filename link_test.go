package multiple_link

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLink_Read(t *testing.T) {
	type fields struct {
		ID               uint32
		sess             *Session
		buf              *bytes.Buffer
		bufLock          sync.Mutex
		readableBufSize  int32
		writeableBufSize int32
		chReadEvent      chan struct{}
		chWriteEvent     chan struct{}
		readDeadline     atomic.Value
		writeDeadline    atomic.Value
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantN   int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Link{
				ID:               tt.fields.ID,
				sess:             tt.fields.sess,
				buf:              tt.fields.buf,
				bufLock:          tt.fields.bufLock,
				readableBufSize:  tt.fields.readableBufSize,
				writeableBufSize: tt.fields.writeableBufSize,
				chReadEvent:      tt.fields.chReadEvent,
				chWriteEvent:     tt.fields.chWriteEvent,
				readDeadline:     tt.fields.readDeadline,
				writeDeadline:    tt.fields.writeDeadline,
			}
			gotN, err := l.Read(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("Read() gotN = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
