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
	//ErrInvalidProtocol = errors.New("invalid protocol")
	ErrTimeout = errors.New("timeout")
)

type writeRequest struct {
	packet Packet
	result chan writeResult
}

type writeResult struct {
	n   int
	err error
}

type Session struct {
	conn   net.Conn
	config *Config

	links    map[uint32]*Link
	linkLock sync.Mutex

	linkID uint32

	chAccepts chan *Link

	//chProtoErr     chan struct{}
	//protoErrorOnce sync.Once

	chSocketReadError   chan struct{}
	socketReadError     atomic.Value
	SocketReadErrorOnce sync.Once

	chSocketWriteError   chan struct{}
	socketWriteError     atomic.Value
	SocketWriteErrorOnce sync.Once

	writes chan writeRequest

	deadline atomic.Value
}

func newSession(config *Config, conn net.Conn) *Session {
	s := new(Session)
	s.config = config
	s.conn = conn
	s.links = make(map[uint32]*Link)
	s.chAccepts = make(chan *Link, defaultAcceptBacklog)
	//s.chProtoErr = make(chan struct{})
	s.chSocketReadError = make(chan struct{})
	s.chSocketWriteError = make(chan struct{})

	go s.readLoop()

	return s
}

func (s *Session) AcceptLink() (*Link, error) {
	var deadline chan time.Time
	// TODO

	select {
	case link := <-s.chAccepts:
		return link, nil
	//case <-s.chProtoErr:
	//	return nil, ErrInvalidProtocol
	case <-s.chSocketReadError:
		return nil, s.socketReadError.Load().(error)
	case <-deadline:
		return nil, ErrTimeout
	}
}

func (s *Session) OpenLink() (*Link, error) {
	link := newLink(atomic.AddUint32(&s.linkID, 1), s)

	newF := newPacket(byte(s.config.Version), cmdSYN, link.ID)
	if _, err := s.writePacket(newF); err != nil {
		return nil, err
	}
	s.linkLock.Lock()
	defer s.linkLock.Unlock()

	select {
	case <-s.chSocketWriteError:
		return nil, s.socketWriteError.Load().(error)
		// TODO other case
	default:
		s.links[link.ID] = link
	}

	return link, nil
}

func (s *Session) readLoop() {
	for {
		// TODO

		if p, err := UnmarshalPacket(s.conn); err == nil {
			pid := p.pid

			switch p.cmd {
			case cmdSYN:
				s.linkLock.Lock()
				if _, ok := s.links[pid]; !ok {
					link := newLink(p.pid, s)
					s.links[pid] = link

					select {
					case s.chAccepts <- link:
						// TODO other case
					}
				}
				s.linkLock.Unlock()

			case cmdPSH:
				s.linkLock.Lock()
				if link, ok := s.links[pid]; ok {
					link.bufLock.Lock()
					link.buf.Write(p.data) // write too large?
					link.bufLock.Unlock()

					atomic.AddInt32(&link.readableBufSize, -int32(len(p.data)))
					link.notifyReadEvent()
				} else {
					// TODO send close back
				}
				s.linkLock.Unlock()

			case cmdPing:
				// TODO keepalive

			case cmdClose:
				s.linkLock.Lock()
				if link, ok := s.links[pid]; ok {
					// TODO how read the remaining data of this link?
					// TODO how peer delete too?
					delete(s.links, pid)
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

func (s *Session) writeLoop() {
	for {
		select {
		// TODO other
		case req := <-s.writes:
			b := MarshalPacket(req.packet)
			n, err := s.conn.Write(b)

			result := writeResult{
				n:   n,
				err: err,
			}

			req.result <- result
			close(req.result)

			if err != nil {
				s.notifyWriteError(err)
				return
			}
		}
	}
}

func (s *Session) writePacket(p Packet) (int, error) {
	req := writeRequest{
		packet: p,
		result: make(chan writeResult, 1),
	}

	select {
	case s.writes <- req:
	case <-s.chSocketWriteError:
		return 0, s.socketWriteError.Load().(error)
	}

	select {
	case result := <-req.result:
		return result.n, result.err
	case <-s.chSocketWriteError:
		return 0, s.socketWriteError.Load().(error)
	}
}

//func (s *Session) notifyProtoErr() {
//	s.protoErrorOnce.Do(func() {
//		close(s.chProtoErr)
//	})
//}

func (s *Session) notifyReadError(err error) {
	s.SocketReadErrorOnce.Do(func() {
		s.socketReadError.Store(err)
		close(s.chSocketReadError)
	})
}

func (s *Session) notifyWriteError(err error) {
	s.SocketWriteErrorOnce.Do(func() {
		s.socketWriteError.Store(err)
		close(s.chSocketWriteError)
	})
}

func (s *Session) SetDeadline(t time.Time) {
	s.deadline.Store(t)
}
