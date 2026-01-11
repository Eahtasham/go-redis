package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func send(conn net.Conn, reader *bufio.Reader, label, cmd string) {
	fmt.Printf("\n[%s] Sending:\n%s", label, cmd)
	_, err := conn.Write([]byte(cmd))
	if err != nil {
		fmt.Println("Write error:", err)
		return
	}

	resp, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}
	fmt.Printf("[%s] Response: %s", label, resp)
}

func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	fmt.Println("Connected to Redis")

	// 1. Simple String (+)
	send(conn, reader, "Simple String",
		"*1\r\n$4\r\nPING\r\n")

	// 2. Error (-)
	send(conn, reader, "Error",
		"*1\r\n$7\r\nINVALID\r\n")

	// 3. Integer (:)
	send(conn, reader, "Integer",
		"*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")

	// 4. Bulk String ($)
	send(conn, reader, "Bulk String",
		"*2\r\n$3\r\nGET\r\n$7\r\ncounter\r\n")

	// 5. Array (*)
	send(conn, reader, "Array",
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n")
}
