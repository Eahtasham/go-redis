package netlayer

import (
	"bufio"
	"fmt"
	"net"
	"sync/atomic"
)

type ClientConn struct {
	conn   net.Conn
	Closed atomic.Bool
}

func HandleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Protocl layer will be implemented here
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		fmt.Println(line)
		writer.WriteString("+OK/r/n")
		writer.Flush()
	}

}
