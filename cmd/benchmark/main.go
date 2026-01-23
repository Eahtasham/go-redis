package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

var (
	host      = flag.String("h", "localhost", "Server hostname")
	port      = flag.Int("p", 6379, "Server port")
	clients   = flag.Int("c", 50, "Number of parallel connections")
	requests  = flag.Int("n", 10000, "Total number of requests")
	dataSize  = flag.Int("d", 3, "Data size in bytes for SET value")
	testType  = flag.String("t", "all", "Test type: set, get, incr, lpush, sadd, all")
	keepAlive = flag.Bool("k", true, "Use keep-alive connections")
)

type BenchResult struct {
	Name       string
	TotalOps   int64
	Duration   time.Duration
	OpsPerSec  float64
	AvgLatency time.Duration
}

func main() {
	flag.Parse()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║              go-redis Benchmark Tool                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Printf("\nServer: %s:%d\n", *host, *port)
	fmt.Printf("Clients: %d, Requests: %d, Data size: %d bytes\n\n", *clients, *requests, *dataSize)

	// Generate value of specified size
	value := make([]byte, *dataSize)
	for i := range value {
		value[i] = 'x'
	}
	valueStr := string(value)

	results := []BenchResult{}

	switch *testType {
	case "set":
		results = append(results, runBenchmark("SET", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("key:%d", id)
			sendCommand(w, r, "SET", key, valueStr)
		}))
	case "get":
		// Pre-populate keys
		setupBenchmark("GET", valueStr)
		results = append(results, runBenchmark("GET", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("key:%d", id%1000)
			sendCommand(w, r, "GET", key)
		}))
	case "incr":
		results = append(results, runBenchmark("INCR", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("counter:%d", id%100)
			sendCommand(w, r, "INCR", key)
		}))
	case "lpush":
		results = append(results, runBenchmark("LPUSH", func(id int, w *resp.Writer, r *resp.Reader) {
			sendCommand(w, r, "LPUSH", "mylist", valueStr)
		}))
	case "sadd":
		results = append(results, runBenchmark("SADD", func(id int, w *resp.Writer, r *resp.Reader) {
			member := fmt.Sprintf("member:%d", id)
			sendCommand(w, r, "SADD", "myset", member)
		}))
	case "all":
		results = append(results, runBenchmark("PING", func(id int, w *resp.Writer, r *resp.Reader) {
			sendCommand(w, r, "PING")
		}))
		results = append(results, runBenchmark("SET", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("key:%d", id)
			sendCommand(w, r, "SET", key, valueStr)
		}))
		setupBenchmark("GET", valueStr)
		results = append(results, runBenchmark("GET", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("key:%d", id%1000)
			sendCommand(w, r, "GET", key)
		}))
		results = append(results, runBenchmark("INCR", func(id int, w *resp.Writer, r *resp.Reader) {
			key := fmt.Sprintf("counter:%d", id%100)
			sendCommand(w, r, "INCR", key)
		}))
		results = append(results, runBenchmark("LPUSH", func(id int, w *resp.Writer, r *resp.Reader) {
			sendCommand(w, r, "LPUSH", "benchlist", valueStr)
		}))
		results = append(results, runBenchmark("SADD", func(id int, w *resp.Writer, r *resp.Reader) {
			member := fmt.Sprintf("member:%d", id)
			sendCommand(w, r, "SADD", "benchset", member)
		}))
	default:
		fmt.Println("Unknown test type:", *testType)
		return
	}

	// Print summary
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                      SUMMARY                               ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-10s │ %12s │ %12s │ %12s ║\n", "Command", "Ops/sec", "Avg Latency", "Total Time")
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	for _, r := range results {
		fmt.Printf("║ %-10s │ %10.0f/s │ %10.2fµs │ %10.2fs ║\n",
			r.Name, r.OpsPerSec, float64(r.AvgLatency.Microseconds()), r.Duration.Seconds())
	}
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
}

func setupBenchmark(name, value string) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Printf("Setup failed: %v\n", err)
		return
	}
	defer conn.Close()

	w := resp.NewWriter(conn)
	r := resp.NewReader(conn)

	// Pre-populate 1000 keys for GET benchmark
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		sendCommand(w, r, "SET", key, value)
	}
}

func runBenchmark(name string, operation func(id int, w *resp.Writer, r *resp.Reader)) BenchResult {
	fmt.Printf("Running %s benchmark...\n", name)

	var totalOps int64
	var totalLatency int64
	var wg sync.WaitGroup

	opsPerClient := *requests / *clients

	start := time.Now()

	for c := 0; c < *clients; c++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
			if err != nil {
				fmt.Printf("Client %d failed to connect: %v\n", clientID, err)
				return
			}
			defer conn.Close()

			w := resp.NewWriter(conn)
			r := resp.NewReader(conn)

			for i := 0; i < opsPerClient; i++ {
				opID := clientID*opsPerClient + i
				opStart := time.Now()
				operation(opID, w, r)
				atomic.AddInt64(&totalLatency, int64(time.Since(opStart)))
				atomic.AddInt64(&totalOps, 1)
			}
		}(c)
	}

	wg.Wait()
	duration := time.Since(start)

	ops := atomic.LoadInt64(&totalOps)
	latency := atomic.LoadInt64(&totalLatency)

	result := BenchResult{
		Name:       name,
		TotalOps:   ops,
		Duration:   duration,
		OpsPerSec:  float64(ops) / duration.Seconds(),
		AvgLatency: time.Duration(latency / ops),
	}

	fmt.Printf("  %s: %.0f ops/sec (avg latency: %.2fµs)\n",
		name, result.OpsPerSec, float64(result.AvgLatency.Microseconds()))

	return result
}

func sendCommand(w *resp.Writer, r *resp.Reader, args ...string) resp.Value {
	vals := make([]resp.Value, len(args))
	for i, arg := range args {
		vals[i] = resp.BulkValue(arg)
	}
	w.WriteValue(resp.ArrayValue(vals))
	response, _ := r.ReadValue()
	return response
}
