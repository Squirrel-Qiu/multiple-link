package multiple_link

import (
	"bytes"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/xerrors"
)

const ACKPayloadLength = 4

type Link struct {
	ID   uint32
	sess *Session

	buf     *bytes.Buffer
	bufLock sync.Mutex

	// the buffer can read or write, control each other's read and write rate
	readableBufSize  int32
	writeableBufSize int32

	chReadEvent  chan struct{}
	chWriteEvent chan struct{}

	readDeadline  atomic.Value
	writeDeadline atomic.Value
}

func newLink(id uint32, sess *Session) *Link {
	link := &Link{
		ID:          id,
		sess:        sess,
		buf:         bytes.NewBuffer(make([]byte, 0)),
		chReadEvent: make(chan struct{}, 1),
		chWriteEvent: make(chan struct{}, 1),
	}

	return link
}

func (l *Link) Read(b []byte) (n int, err error) {
	var deadline <-chan time.Time
	if d, ok := l.readDeadline.Load().(time.Time); ok && !d.IsZero() {
		timer := time.NewTimer(time.Until(d))
		defer timer.Stop()
		deadline = timer.C
	}

	if len(b) == 0 {
		return 0, nil
	}

	for {
		l.bufLock.Lock()
		n, err = l.buf.Read(b)
		l.bufLock.Unlock()

		if n > 0 {
			atomic.AddInt32(&l.readableBufSize, int32(n))

			select {
			// TODO when this link close
			default:
				go l.sendACK()
			}

			return
		}

		select {
		case <-l.chReadEvent:
		// readEvent from readLoop(peer)

		case <-l.sess.chSocketReadError:
			return 0, l.sess.socketReadError.Load().(error)
		case <-deadline:
			return 0, xerrors.Errorf("link read failed: %w", ErrTimeout)
		}
	}
}

func (l *Link) Write(b []byte) (n int, err error) {
	var deadline <-chan time.Time
	if d, ok := l.writeDeadline.Load().(time.Time); ok && !d.IsZero() {
		timer := time.NewTimer(time.Until(d))
		defer timer.Stop()
		deadline = timer.C
	}

	if len(b) == 0 {
		return 0, err
	}

	p := newPacket(byte(l.sess.config.Version), cmdPSH, l.ID)
	p.data = b

	select {
	case <-deadline:
		return 0, xerrors.Errorf("link write failed: %w", err)
	// TODO other case (die)
	// read ACK packet, update writeableBufSize and then notify
	case <-l.chWriteEvent:
		err = l.sess.writePacket(p)
		if err != nil {
			return 0, xerrors.Errorf("link write failed: %w", err)
		}

		if atomic.AddInt32(&l.writeableBufSize, -int32(len(b))) > 0 {
			l.notifyWriteEvent()
		}

		return len(b), nil
	}
}

func (l *Link) sendACK() {
	p := newPacket(byte(l.sess.config.Version), cmdACK, l.ID)

	b := make([]byte, ACKPayloadLength)
	size := atomic.LoadInt32(&l.readableBufSize)
	if size < 0 {
		size = 0
	}
	binary.BigEndian.PutUint32(b, uint32(size))
	p.data = b
	_ = l.sess.writePacket(p)
}

func (l *Link) notifyReadEvent() {
	select {
	case l.chReadEvent <- struct{}{}:
	default:
	}
}

func (l *Link) notifyWriteEvent() {
	select {
	case l.chWriteEvent <- struct{}{}:
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
