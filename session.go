package multiple_link

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/xerrors"
)

const defaultAcceptBacklog = 1024

var (
	//ErrInvalidProtocol = errors.New("invalid protocol")
	ErrTimeout = errors.New("timeout")
)

type writeRequest struct {
	packet  *Packet
	written chan struct{}
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

	closeOnce sync.Once
	die chan struct{}
}

func NewSession(config *Config, conn net.Conn) *Session {
	s := new(Session)
	s.config = config
	s.conn = conn
	s.links = make(map[uint32]*Link)
	s.chAccepts = make(chan *Link, defaultAcceptBacklog)
	//s.chProtoErr = make(chan struct{})
	s.chSocketReadError = make(chan struct{})
	s.chSocketWriteError = make(chan struct{})
	s.writes = make(chan writeRequest)

	go s.readLoop()
	go s.writeLoop()

	return s
}

func (s *Session) AcceptLink() (*Link, error) {
	fmt.Println(s.config.Mode, "start accept link")
	var deadline <-chan time.Time
	if d, ok := s.deadline.Load().(time.Time); ok && !d.IsZero() {
		timer := time.NewTimer(time.Until(d))
		defer timer.Stop()
		deadline = timer.C
	}

	select {
	case link := <-s.chAccepts:
		fmt.Println(s.config.Mode, " accept link from chAccepts")
		return link, nil
	//case <-s.chProtoErr:
	//	return nil, ErrInvalidProtocol
	case <-s.chSocketReadError:
		return nil, s.socketReadError.Load().(error)
	case <-deadline:
		return nil, ErrTimeout
	case <-s.die:
		return nil, io.ErrClosedPipe
	}
}

func (s *Session) OpenLink() (*Link, error) {
	link := newLink(atomic.AddUint32(&s.linkID, 1), s)
	fmt.Println(s.config.Mode, " open link, link ID:", link.ID)

	newP := newPacket(byte(s.config.Version), cmdSYN, link.ID)
	if err := s.writePacket(newP); err != nil {
		return nil, err
	}
	s.linkLock.Lock()
	defer s.linkLock.Unlock()

	select {
	case <-s.die:
		return nil, io.ErrClosedPipe
	case <-s.chSocketWriteError:
		return nil, s.socketWriteError.Load().(error)
		// TODO other case(die)
	default:
		s.links[link.ID] = link
		fmt.Println(s.config.Mode, " map add link ok")
	}

	return link, nil
}

func (s *Session) readLoop() {
	for {
		select {
		case <-s.die:
			return
		default:
		}

		if p, err := UnmarshalPacket(s.conn); err == nil {
			fmt.Println(s.config.Mode, " finish unmarshal packet")
			pid := p.pid

			switch p.cmd {
			case cmdSYN:
				s.linkLock.Lock()
				if _, ok := s.links[pid]; !ok {
					link := newLink(p.pid, s)
					s.links[pid] = link

					select {
					case s.chAccepts <- link:
						fmt.Println(s.config.Mode, " chAccepts accept link")
						// TODO other case
					}
				}
				s.linkLock.Unlock()
			case cmdACK:
				s.linkLock.Lock()
				if link, ok := s.links[pid]; ok {
					atomic.StoreInt32(&link.writeableBufSize, int32(binary.BigEndian.Uint32(p.data)))

					if atomic.LoadInt32(&link.writeableBufSize) > 0 {
						link.notifyWriteEvent()
					}
				}
				s.linkLock.Unlock()
			case cmdPSH:
				fmt.Println(s.config.Mode, " readLoop read PSH-packet: ", len(p.data))
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
					link.closeByPeer()
				} else {
					// TODO send close back
				}
				s.linkLock.Unlock()
			}
		} else {
			s.notifyReadError(err)
			return
		}

	}
}

func (s *Session) writeLoop() {
	fmt.Println(s.config.Mode, " writeLoop:")
	for {
		select {
		case req := <-s.writes:
			fmt.Println(s.config.Mode, " accept s.writes:")
			b := MarshalPacket(req.packet)

			if _, err := s.conn.Write(b); err != nil {
				s.closeOnce.Do(func() {
					close(s.die)
				})

				s.notifyWriteError(err)
				return
			}

			close(req.written)
		case <-s.die:
			return
		}
	}
}

func (s *Session) writePacket(p *Packet) error {
	req := writeRequest{
		packet:  p,
		written: make(chan struct{}),
	}

	select {
	case s.writes <- req:
		fmt.Println(s.config.Mode, " write packet to s.writes, cmd is", p.cmd)

	case <-s.chSocketWriteError:
		return s.socketWriteError.Load().(error)
	}

	select {
	case <-req.written:
		return nil

	case <-s.chSocketWriteError:
		return s.socketWriteError.Load().(error)
	}
}

func (s *Session) removeLink(id uint32) {
	delete(s.links, id)
}

func (s *Session) Close() (err error) {
	fmt.Println(s.config.Mode, " close===============")
	s.closeOnce.Do(func() {
		close(s.die)

		s.linkLock.Lock()
		for id := range s.links {
			s.removeLink(id)

			err = s.writePacket(newPacket(byte(s.config.Version), cmdClose, id))
		}
		s.linkLock.Unlock()

		// TODO close peer session?
	})

	if err != nil {
		return xerrors.Errorf("session close failed: %w", err)
	}

	return s.conn.Close()
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
