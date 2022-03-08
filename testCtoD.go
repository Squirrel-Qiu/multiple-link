package multiple_link

import (
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/xerrors"
)

func testCtoD() {
	listener2, err := net.Listen("tcp", "127.0.0.1:6060")
	if err != nil {
		panic(err)
	}

	for {
		ListenC, err := listener2.Accept()
		if err != nil {
			log.Printf("accept connection failed: %v", err)
			continue
		}

		go func() {
			serverConf := DefaultConfig(ServerMode)
			server := NewSession(serverConf, ListenC)

			for {
				link, err := server.AcceptLink()
				if err != nil {
					log.Printf("%+v", xerrors.Errorf("server accept link failed: %w", err))
					return
				}

				go func() {
					defer link.Close()

					dialC, err := net.Dial("tcp", "127.0.0.1:7070")
					if err != nil {
						log.Printf("dial to D failed: %v", err)
						return
					}
					fmt.Println("C dial to D ok")

					go func() {

						if _, err := io.Copy(dialC, link); err != nil {
							log.Printf("C copy to D failed: %v", err)
						}

						link.Close()
						dialC.Close()
					}()

					if _, err := io.Copy(link, dialC); err != nil {
						log.Printf("D copy to C failed: %v", err)
					}

					dialC.Close()
				}()
			}
		}()
	}
}
