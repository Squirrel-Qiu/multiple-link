package multiple_link

import (
	"bytes"
	"sync"
	"sync/atomic"
)

type Link struct {
	ID   uint32
	sess *Session

	buf             *bytes.Buffer
	readableBufSize int32  // the buffer can read
	bufLock         sync.Mutex

	chReadEvent chan struct{}
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

		case <-l.sess.chSocketReadError:
			return 0, l.sess.socketReadError.Load().(error)
			// TODO other case (timeout)
		}
	}
}

func (l *Link) notifyReadEvent() {
	select {
	case l.chReadEvent <- struct{}{}:
	default:
	}
}
