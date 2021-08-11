package multiple_link

import (
	"sync"
)

type Link struct {
	ID uint32
	sess *Session
	buf [][]byte
	bufLock sync.Mutex
	readEvent chan struct{}
}

func newLink(id uint32, sess *Session) *Link {
	link := new(Link)
	link.ID = id
	link.sess = sess

	return link
}

func (link *Link) Read(b []byte) (n int, err error) {
	return 0, nil
}
