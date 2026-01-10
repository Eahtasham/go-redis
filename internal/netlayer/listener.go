package netlayer

import (
	"context"
	"net"
	"sync"
)

type Listener struct {
	ln net.Listener
	wg sync.WaitGroup
}

func Newlistener(addr string) (*Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

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
