package netlayer

import (
	"fmt"
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

	for {
		// Protocl layer will be implemented here
		line, err := reader.ReadValue()
		if err != nil {
			return
		}
		fmt.Println(line)
		res := commands.Dispatch(line)
		writer.WriteValue(res)
	}

}
