package main

import (
	"fmt"
	"net"
	"os"
	"time"

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
	}
	return "unknown"
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

	fmt.Println("=== Active Expiration Test ===")
	fmt.Println()

	// Test 1: Set a key with 2 second TTL
	fmt.Println("1. Setting key 'expiring' with value 'hello' and 2s TTL")
	sendCommand(writer, reader, "SET", "expiring", "hello")
	sendCommand(writer, reader, "EXPIRE", "expiring", "2")

	// Verify it exists
	fmt.Printf("   GET expiring -> %s\n", formatResponse(sendCommand(writer, reader, "GET", "expiring")))
	fmt.Printf("   TTL expiring -> %s\n", formatResponse(sendCommand(writer, reader, "TTL", "expiring")))

	// Wait for expiration (2.5 seconds to be safe)
	fmt.Println()
	fmt.Println("2. Waiting 2.5 seconds for key to expire...")
	time.Sleep(2500 * time.Millisecond)

	// Check if key was removed by active expiration
	fmt.Println()
	fmt.Println("3. Checking if key was expired by active sweeper:")
	result := sendCommand(writer, reader, "GET", "expiring")
	fmt.Printf("   GET expiring -> %s\n", formatResponse(result))

	if result.Type == resp.BulkString && result.Str == "" {
		fmt.Println()
		fmt.Println("   PASS: Key was actively expired!")
	} else {
		fmt.Println()
		fmt.Println("   FAIL: Key still exists (shouldn't happen)")
	}

	// Test 2: Multiple keys with different TTLs
	fmt.Println()
	fmt.Println("4. Setting multiple keys with 1s TTL...")
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("temp%d", i)
		sendCommand(writer, reader, "SET", key, "value")
		sendCommand(writer, reader, "EXPIRE", key, "1")
	}
	fmt.Println("   Created 10 keys with 1s TTL")

	fmt.Println()
	fmt.Println("5. Waiting 1.5 seconds...")
	time.Sleep(1500 * time.Millisecond)

	// Check how many keys remain
	fmt.Println()
	fmt.Println("6. Checking if keys were expired:")
	expiredCount := 0
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("temp%d", i)
		result := sendCommand(writer, reader, "GET", key)
		if result.Type == resp.BulkString && result.Str == "" {
			expiredCount++
		}
	}
	fmt.Printf("   %d/10 keys expired by active sweeper\n", expiredCount)
	if expiredCount == 10 {
		fmt.Println("   PASS: All keys expired!")
	}

	fmt.Println()
	fmt.Println("=== Test Complete ===")
}
