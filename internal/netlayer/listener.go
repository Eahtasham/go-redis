package netlayer

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type Listener struct {
	ln net.Listener
	wg sync.WaitGroup
}

func NewListener(addr string) (*Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// panic(err)
		return nil, err
	}
	fmt.Println("listening on", addr)

	return &Listener{ln: ln}, nil
}

func (l *Listener) Serve(ctx context.Context, handler func(net.Conn)) error {
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}

		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			handler(conn)
		}()

	}
}
