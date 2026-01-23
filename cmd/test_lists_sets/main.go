package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

func sendCommand(writer *resp.Writer, reader *resp.Reader, args ...string) resp.Value {
	vals := make([]resp.Value, len(args))
	for i, arg := range args {
		vals[i] = resp.BulkValue(arg)
	}
	writer.WriteValue(resp.ArrayValue(vals))
	response, _ := reader.ReadValue()
	return response
}

func formatResponse(v resp.Value) string {
	switch v.Type {
	case resp.SimpleString:
		return fmt.Sprintf("+%s", v.Str)
	case resp.Error:
		return fmt.Sprintf("-%s", v.Str)
	case resp.Integer:
		return fmt.Sprintf(":%d", v.Int)
	case resp.BulkString:
		if v.Str == "" {
			return "(nil)"
		}
		return fmt.Sprintf("\"%s\"", v.Str)
	case resp.Array:
		if len(v.Array) == 0 {
			return "(empty array)"
		}
		result := fmt.Sprintf("[%d] ", len(v.Array))
		for i, item := range v.Array {
			if i > 0 {
				result += ", "
			}
			result += formatResponse(item)
		}
		return result
	}
	return "unknown"
}

func test(writer *resp.Writer, reader *resp.Reader, label string, args ...string) {
	result := sendCommand(writer, reader, args...)
	fmt.Printf("  %s -> %s\n", label, formatResponse(result))
}

func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Failed to connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)

	fmt.Println("=== List & Set Commands Test ===")

	// Clean up first
	sendCommand(writer, reader, "DEL", "mylist", "myset", "set1", "set2")

	// List tests
	fmt.Println("\n--- LIST COMMANDS ---")
	test(writer, reader, "RPUSH mylist a b c", "RPUSH", "mylist", "a", "b", "c")
	test(writer, reader, "LPUSH mylist z", "LPUSH", "mylist", "z")
	test(writer, reader, "LRANGE mylist 0 -1", "LRANGE", "mylist", "0", "-1")
	test(writer, reader, "LLEN mylist", "LLEN", "mylist")
	test(writer, reader, "LINDEX mylist 0", "LINDEX", "mylist", "0")
	test(writer, reader, "LINDEX mylist -1", "LINDEX", "mylist", "-1")
	test(writer, reader, "LPOP mylist", "LPOP", "mylist")
	test(writer, reader, "RPOP mylist", "RPOP", "mylist")
	test(writer, reader, "LRANGE mylist 0 -1", "LRANGE", "mylist", "0", "-1")

	// Set tests
	fmt.Println("\n--- SET COMMANDS ---")
	test(writer, reader, "SADD myset a b c", "SADD", "myset", "a", "b", "c")
	test(writer, reader, "SADD myset c d", "SADD", "myset", "c", "d")
	test(writer, reader, "SMEMBERS myset", "SMEMBERS", "myset")
	test(writer, reader, "SISMEMBER myset a", "SISMEMBER", "myset", "a")
	test(writer, reader, "SISMEMBER myset x", "SISMEMBER", "myset", "x")
	test(writer, reader, "SCARD myset", "SCARD", "myset")
	test(writer, reader, "SREM myset a", "SREM", "myset", "a")
	test(writer, reader, "SMEMBERS myset", "SMEMBERS", "myset")

	// Set operations
	fmt.Println("\n--- SET OPERATIONS ---")
	sendCommand(writer, reader, "SADD", "set1", "a", "b", "c")
	sendCommand(writer, reader, "SADD", "set2", "b", "c", "d")
	test(writer, reader, "SUNION set1 set2", "SUNION", "set1", "set2")
	test(writer, reader, "SINTER set1 set2", "SINTER", "set1", "set2")

	fmt.Println("\n=== All Tests Complete ===")
}
