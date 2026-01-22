package netlayer

import (
	"io"
	"net"
	"sync/atomic"

	"github.com/Eahtasham/go-redis/internal/commands"
	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

type ClientConn struct {
	conn   net.Conn
	Closed atomic.Bool
}

func HandleConn(conn net.Conn) {
	defer conn.Close()

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	// Per-client context for transactions
	ctx := &commands.ClientContext{}

	for {
		value, err := reader.ReadValue()
		if err != nil {
			if err != io.EOF {
				// Log error if needed
			}
			return
		}

		res := commands.DispatchWithContext(value, ctx)
		writer.WriteValue(res)
	}
}
