package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	// Send GET counter command
	writer.WriteValue(resp.ArrayValue([]resp.Value{
		resp.BulkValue("GET"),
		resp.BulkValue("counter"),
	}))

	response, _ := reader.ReadValue()

	fmt.Println("=== AOF Replay Verification ===")
	fmt.Printf("GET counter -> ")
	if response.Type == resp.BulkString && response.Str == "3" {
		fmt.Printf("\"%s\" (PASS - data persisted!)\n", response.Str)
	} else {
		fmt.Printf("%v (FAIL - expected \"3\")\n", response)
	}

	// Also check mykey was deleted
	writer.WriteValue(resp.ArrayValue([]resp.Value{
		resp.BulkValue("GET"),
		resp.BulkValue("mykey"),
	}))

	response, _ = reader.ReadValue()
	fmt.Printf("GET mykey -> ")
	if response.Type == resp.BulkString && response.Str == "" {
		fmt.Printf("(nil) (PASS - delete persisted!)\n")
	} else {
		fmt.Printf("%v (FAIL - expected nil)\n", response)
	}
}
