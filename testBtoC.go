package multiple_link

import (
	"fmt"
	"io"
	"log"
	"net"
)

func testBtoC() {
	dialC, err := net.Dial("tcp", "127.0.0.1:6060")
	if err != nil {
		log.Printf("dial to C failed: %v", err)
		return
	}
	fmt.Println("B dial to C ok")

	clientConf := DefaultConfig(ClientMode)
	client := NewSession(clientConf, dialC)

	listenerB, err := net.Listen("tcp", "127.0.0.1:5050")
	if err != nil {
		panic(err)
	}

	for {
		accept, err := listenerB.Accept()
		if err != nil {
			log.Printf("accept connection failed: %v", err)
			continue
		}

		go func() {
			link, err := client.OpenLink()
			if err != nil {
				log.Printf("client open link failed: %v", err)

				return
			}

			defer accept.Close()
			defer link.Close()

			go func() {
				defer accept.Close()
				defer link.Close()

				if _, err := io.Copy(link, accept); err != nil {
					log.Printf("A copy to B failed: %v", err)
				}
			}()

			if _, err := io.Copy(accept, link); err != nil {
				log.Printf("B copy to A failed: %v", err)
			}
		}()

	}
}
