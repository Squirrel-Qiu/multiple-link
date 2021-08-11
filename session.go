package multiple_link

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const defaultAcceptBacklog = 1024

var (
	ErrInvalidProtocol = errors.New("invalid protocol")
	ErrTimeout         = errors.New("timeout")
)

type Session struct {
	conn   net.Conn
	config *Config

	links    map[uint32]*Link
	linkLock sync.Mutex

	chAccepts chan *Link

	chProtoErr     chan struct{}
	protoErrorOnce sync.Once

	chSocketReadError chan struct{}
	socketReadError atomic.Value
	SocketReadErrorOnce sync.Once

	deadline atomic.Value
}

func newSession(config *Config, conn net.Conn) *Session {
	s := new(Session)
	s.config = config
	s.conn = conn
	s.links = make(map[uint32]*Link)
	s.chAccepts = make(chan *Link, defaultAcceptBacklog)
	s.chProtoErr = make(chan struct{})
	s.chSocketReadError = make(chan struct{})

	go s.readLoop()

	return s
}

func (s *Session) AcceptLink() (*Link, error) {
	var deadline chan time.Time
	// TODO

	select {
	case link := <-s.chAccepts:
		return link, nil
	case <-s.chProtoErr:
		return nil, ErrInvalidProtocol
	case <-s.chSocketReadError:
		return nil, s.socketReadError.Load().(error)
	case <-deadline:
		return nil, ErrTimeout
	}
}

func (s *Session) readLoop() {
	for {
		// TODO

		if f, err := UnmarshalFrame(s.conn); err == nil {
			if f.ver != byte(s.config.Version) {
				s.notifyProtoErr()
			}

			fid := f.fid

			switch f.cmd {
			case cmdSYN:
				s.linkLock.Lock()
				if _, ok := s.links[fid]; !ok {
					link := newLink(f.fid, s)
					s.links[fid] = link

					select {
					case s.chAccepts <- link:
						//
					}
				}
				s.linkLock.Unlock()
			case cmdPSH:
				s.linkLock.Lock()
				if link, ok := s.links[fid]; ok {
					link.bufLock.Lock()
					link.buf = append(link.buf, f.data)
					link.bufLock.Unlock()
				} else {
					// TODO send close back
				}
				s.linkLock.Unlock()
			case cmdPing:
				// TODO keepalive
			case cmdClose:
				s.linkLock.Lock()
				if link, ok := s.links[fid]; ok {
					// TODO how peer delete too?
					delete(s.links, fid)
					link.closebypeer()
				} else {
					// TODO send close back
				}
				s.linkLock.Unlock()
			default:
				return
			}
		} else {
			s.notifyReadError(err)
		}

	}
}

func (s *Session) notifyProtoErr() {
	s.protoErrorOnce.Do(func() {
		close(s.chProtoErr)
	})
}

func (s *Session) notifyReadError(err error) {
	s.SocketReadErrorOnce.Do(func() {
		s.socketReadError.Store(err)
		close(s.chSocketReadError)
	})
}

func (s *Session) SetDeadline(t time.Time) {
	s.deadline.Store(t)
}
