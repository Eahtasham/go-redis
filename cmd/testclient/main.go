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

	err := writer.WriteValue(resp.ArrayValue(vals))
	if err != nil {
		return resp.ErrorValue(fmt.Sprintf("Write error: %v", err))
	}

	response, err := reader.ReadValue()
	if err != nil {
		return resp.ErrorValue(fmt.Sprintf("Read error: %v", err))
	}
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
		return fmt.Sprintf("[%d items]", len(v.Array))
	}
	return "unknown"
}

func test(writer *resp.Writer, reader *resp.Reader, expected string, args ...string) {
	result := sendCommand(writer, reader, args...)
	actual := formatResponse(result)
	status := "PASS"
	if expected != "" && actual != expected {
		status = "FAIL"
	}
	fmt.Printf("[%s] %v -> %s\n", status, args, actual)
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

	fmt.Println("go-redis Integration Test")
	fmt.Println("=========================")

	test(writer, reader, "+PONG", "PING")
	test(writer, reader, "\"hello\"", "PING", "hello")
	test(writer, reader, "+OK", "SET", "mykey", "myvalue")
	test(writer, reader, "\"myvalue\"", "GET", "mykey")
	test(writer, reader, ":1", "EXISTS", "mykey")
	test(writer, reader, ":1", "INCR", "counter")
	test(writer, reader, ":2", "INCR", "counter")
	test(writer, reader, ":3", "INCR", "counter")
	test(writer, reader, "\"3\"", "GET", "counter")
	test(writer, reader, ":1", "DEL", "mykey")
	test(writer, reader, "(nil)", "GET", "mykey")
	test(writer, reader, "", "FOOBAR") // Should be error

	// Transaction test
	fmt.Println("\nTransaction Test:")
	test(writer, reader, "+OK", "MULTI")
	test(writer, reader, "+QUEUED", "SET", "tx1", "value1")
	test(writer, reader, "+QUEUED", "SET", "tx2", "value2")
	test(writer, reader, "[2 items]", "EXEC")
	test(writer, reader, "\"value1\"", "GET", "tx1")
	test(writer, reader, "\"value2\"", "GET", "tx2")

	fmt.Println("\nAll tests completed!")
}
