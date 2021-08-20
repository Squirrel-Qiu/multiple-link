package multiple_link

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/xerrors"
)

type Link struct {
	ID   uint32
	sess *Session

	buf             *bytes.Buffer
	readableBufSize int32  // the buffer can read
	bufLock         sync.Mutex

	chReadEvent chan struct{}

	readDeadline atomic.Value
	writeDeadline atomic.Value
}

func newLink(id uint32, sess *Session) *Link {
	link := &Link{
		ID:          id,
		sess:        sess,
		buf:         bytes.NewBuffer(make([]byte, 0)),
		chReadEvent: make(chan struct{}, 1),
	}

	return link
}

func (l *Link) Read(b []byte) (n int, err error) {
	// TODO set deadline
	l.readDeadline.Load()

	if len(b) == 0 {
		return 0, nil
	}

	for {
		l.bufLock.Lock()
		n, err = l.buf.Read(b)
		l.bufLock.Unlock()

		if n > 0 {
			atomic.AddInt32(&l.readableBufSize, int32(n))
			return
		}

		select {
		case <-l.chReadEvent:
		// readEvent from readLoop(peer)

		case <-l.sess.chSocketReadError:
			return 0, l.sess.socketReadError.Load().(error)
			// TODO other case (timeout)
		}
	}
}

func (l *Link) Write(b []byte) (n int, err error) {
	l.writeDeadline.Load()

	if len(b) == 0 {
		return 0, err
	}

	p := newPacket(byte(l.sess.config.Version), cmdPSH, l.ID)
	p.data = b
	p.pid = l.ID
	n, err = l.sess.writePacket(p)
	if err != nil {
		return 0, xerrors.Errorf("link write failed: %w", err)
	}
	return len(b), nil
}

func (l *Link) notifyReadEvent() {
	select {
	case l.chReadEvent <- struct{}{}:
	default:
	}
}

func (l *Link) SetDeadline(t time.Time) error {
	l.readDeadline.Store(t)
	l.writeDeadline.Store(t)
	return nil
}

func (l *Link) SetReadDeadline(t time.Time) error {
	l.readDeadline.Store(t)
	return nil
}

func (l *Link) SetWriteDeadline(t time.Time) error {
	l.writeDeadline.Store(t)
	return nil
}
