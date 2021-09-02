package multiple_link

import (
	"io/ioutil"
	"net"
	"sync"
	"testing"

	"golang.org/x/xerrors"
)

func initTest(t *testing.T, addr string) (client net.Conn, server net.Conn) {
	var err error
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		client, err = net.Dial("tcp", addr)
		if err != nil {
			t.Errorf("%+v", xerrors.Errorf("dial connection failed: %w", err))
		}
		wg.Done()
	}()

	go func() {
		listener, _ := net.Listen("tcp", addr)
		server, err = listener.Accept()
		if err != nil {
			t.Errorf("%+v", xerrors.Errorf("accept connection failed: %w", err))
		}
		wg.Done()
	}()

	wg.Wait()
	
	return client, server
}

func TestLinkClientToServer(t *testing.T) {
	clientConn, serverConn := initTest(t, "127.0.0.1:6060")
	defer func() {
		clientConn.Close()
		serverConn.Close()
	}()

	clientConf := DefaultConfig(ClientMode)
	serverConf := DefaultConfig(ServerMode)

	client := NewSession(clientConf,clientConn)
	server := NewSession(serverConf, serverConn)

	file, err := ioutil.ReadFile("testdata/more-than-254-less-than-65535")
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("read file failed:", err))
	}

	go func() {
		link, err := client.OpenLink()
		if err != nil {
			t.Fatalf("%+v", xerrors.Errorf("client open link failed: %w", err))
		}
		defer link.Close()

		if _, err := link.Write(file); err != nil {
			t.Fatalf("%+v", xerrors.Errorf("client link write failed: %w", err))
		}
	}()

	link, err := server.AcceptLink()
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("server accept link failed: %w", err))
	}
	defer link.Close()

	b, err := ioutil.ReadAll(link)  // As of Go 1.16, this function simply calls io.ReadAll.
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("server link read failed: %w", err))
	}

	if string(b) != string(file) {
		t.Errorf("data verify failed, receive:\n%s\n\n\n\nwant:\n%s", string(b), string(file))
	}
}

func TestLinkServerToClient(t *testing.T) {
	clientConn, serverConn := initTest(t, "127.0.0.1:6060")
	defer func() {
		clientConn.Close()
		serverConn.Close()
	}()

	clientConf := DefaultConfig(ClientMode)
	serverConf := DefaultConfig(ServerMode)

	client := NewSession(clientConf,clientConn)
	server := NewSession(serverConf, serverConn)

	file, err := ioutil.ReadFile("testdata/more-than-254-less-than-65535")
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("read file failed:", err))
	}

	go func() {
		link, err := server.OpenLink()
		if err != nil {
			t.Fatalf("%+v", xerrors.Errorf("client open link failed: %w", err))
		}
		defer link.Close()

		if _, err := link.Write(file); err != nil {
			t.Fatalf("%+v", xerrors.Errorf("client link write failed: %w", err))
		}
	}()

	link, err := client.AcceptLink()
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("server accept link failed: %w", err))
	}
	defer link.Close()

	b, err := ioutil.ReadAll(link)
	if err != nil {
		t.Fatalf("%+v", xerrors.Errorf("server link read failed: %w", err))
	}

	if string(b) != string(file) {
		t.Errorf("data verify failed, receive:\n%s\n\n\n\nwant:\n%s", string(b), string(file))
	}
}
