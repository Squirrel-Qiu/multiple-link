package multiple_link

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
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

	closeOnce sync.Once
	die chan struct{}

	eof sync.Once
}

func newLink(id uint32, sess *Session) *Link {
	link := &Link{
		ID:          id,
		sess:        sess,
		buf:         bytes.NewBuffer(make([]byte, 0, defaultBufSize)),
		readableBufSize: sess.config.BufSize,
		writeableBufSize: sess.config.BufSize,
		chReadEvent: make(chan struct{}, 1),
		chWriteEvent: make(chan struct{}, 1),
		die: make(chan struct{}),
	}

	link.notifyWriteEvent()

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
			case <-l.die:
				// when the link is closed, peer doesn't care about the ack because it won't send any packets again
			default:
				go l.sendACK()
			}

			return
		}

		select {
		case <-l.chReadEvent:
		// readEvent from readLoop(peer), continue read and return

		case <-l.sess.chSocketReadError:
			return 0, l.sess.socketReadError.Load().(error)
		case <-deadline:
			return 0, xerrors.Errorf("link read failed: %w", ErrTimeout)
		case <-l.die:
			err = xerrors.Errorf("link read failed: %w", io.ErrClosedPipe)

			l.eof.Do(func() {
				err = io.EOF
			})

			if err == io.EOF {
				return
			}

			return
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
	p.length = uint16(len(b))
	p.data = b

	select {
	case <-deadline:
		return 0, xerrors.Errorf("link write failed: %w", ErrTimeout)
	case <-l.die:
		return 0, xerrors.Errorf("link write failed: %w", io.ErrClosedPipe)
	// read ACK packet, update writeableBufSize, only if writeableBufSize > 0 notify
	case <-l.chWriteEvent:
		log.Println(l.sess.config.Mode, "start write packet")
		err = l.sess.writePacket(p)
		if err != nil {
			return 0, xerrors.Errorf("link write failed: %w", err)
		}

		if atomic.AddInt32(&l.writeableBufSize, -int32(len(b))) > 0 {
			log.Printf("write notify event, writeableBufSize is %v ~~", atomic.LoadInt32(&l.writeableBufSize))
			l.notifyWriteEvent()
		}

		return len(b), nil
	}
}

func (l *Link) Close() (err error) {
	l.closeOnce.Do(func() {
		close(l.die)

		l.sess.removeLink(l.ID)
		log.Printf("close this link, ID is %v", l.ID)

		err = l.sess.writePacket(newPacket(byte(l.sess.config.Version), cmdClose, l.ID))
	})

	if err != nil {
		return xerrors.Errorf("link close failed: %w", err)
	}
	return nil
}

func (l *Link) closeByPeer() {
	log.Printf("close link by peer, ID is %v", l.ID)
	l.closeOnce.Do(func() {
		close(l.die)
	})

	l.sess.removeLink(l.ID)
}

func (l *Link) sendACK() {
	p := newPacket(byte(l.sess.config.Version), cmdACK, l.ID)

	b := make([]byte, ACKPayloadLength)
	size := atomic.LoadInt32(&l.readableBufSize)
	if size < 0 {
		size = 0
	}
	log.Println("sendACK the readableBufSize is ", size)

	binary.BigEndian.PutUint32(b, uint32(size))
	p.length = uint16(len(b))
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
