package multiple_link

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrInvalidProtocol = errors.New("invalid protocol")
)

type Session struct {
	conn net.Conn
	config *Config

	links map[uint32]*Link
	linkLock sync.Mutex
}

func newSession(config *Config, conn net.Conn) *Session {
	s := new(Session)
	s.config = config
	s.conn = conn

	go s.readLoop()

	return s
}

func (s *Session) AcceptLink() (*Link, error) {
}

func (s *Session) readLoop() (err error) {
	for {
		f, err := UnmarshalFrame(s.conn)
		if err != nil {
			return err
		}

		if f.ver != byte(s.config.Version) {
			return errors.New("invalid protocol")
		}

		fid := f.fid

		switch f.cmd {
		case cmdSYN:
			s.linkLock.Lock()
			if _, ok := s.links[fid]; !ok {
				link := newLink(f.fid, s)
				s.links[fid] = link
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
	}
}
