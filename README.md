# 🚀 go-redis

**A Redis clone built from scratch in Go. No dependencies. No magic. Just raw systems programming.**

---

> *"The best way to understand how something works is to build it yourself."*

This project is a learning-focused implementation of Redis. Not a production database—a **deep dive into the guts of how in-memory data stores actually work**.

You'll find:
- How TCP servers handle thousands of concurrent connections
- How the RESP protocol turns bytes into commands
- How expiration works without scanning every key
- How persistence survives server crashes
- How transactions provide atomicity without rollback

**If you've ever wondered what happens between `SET foo bar` and getting `OK` back—this is for you.**

---

## 📖 Table of Contents

- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [Supported Commands](#-supported-commands)
- [Deep Dives](#-deep-dives)
  - [Network Layer](#1-network-layer)
  - [RESP Protocol](#2-resp-protocol)
  - [Command Dispatch](#3-command-dispatch)
  - [In-Memory Store](#4-in-memory-store)
  - [Expiration](#5-expiration-lazy--active)
  - [Persistence](#6-aof-persistence)
  - [Transactions](#7-transactions)
- [Project Structure](#-project-structure)
- [Running Tests](#-running-tests)
- [Benchmarks](#-benchmarks)
- [What's Next](#-whats-next)

---

## ⚡ Quick Start

```bash
# Clone and run
git clone https://github.com/Eahtasham/go-redis.git
cd go-redis
go run ./cmd/server

# In another terminal, use any Redis client
redis-cli -p 6379
> SET mykey "hello"
OK
> GET mykey
"hello"
> INCR counter
(integer) 1
```

Or use the built-in test client:
```bash
go run ./cmd/testclient
```

---

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         TCP Client                               │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Network Layer (netlayer)                      │
│  • TCP Listener on :6379                                         │
│  • Goroutine-per-connection model                                │
│  • Graceful shutdown with context                                │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Protocol Layer (resp)                         │
│  • RESP Reader: bytes → Value (arrays, strings, integers)       │
│  • RESP Writer: Value → bytes                                    │
│  • Zero business logic, pure protocol                            │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                Command Dispatch (commands)                       │
│  • Parse command name + args                                     │
│  • Registry lookup                                               │
│  • Per-client context (for transactions)                         │
│  • MULTI/EXEC queuing                                            │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                 Command Handlers (handlers)                      │
│  PING, SET, GET, DEL, EXISTS, EXPIRE, TTL, INCR, DECR, INCRBY  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
              ┌────────────────┴────────────────┐
              ▼                                 ▼
┌──────────────────────────┐     ┌──────────────────────────────┐
│   In-Memory Store        │     │      AOF Persistence         │
│  • map[string]*Entry     │     │  • Async write pipeline      │
│  • RWMutex protection    │     │  • RESP-encoded commands     │
│  • Lazy + Active expiry  │     │  • Replay on startup         │
└──────────────────────────┘     └──────────────────────────────┘
```

---

## 💻 Supported Commands

### String Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `PING` | `PING [message]` | Returns PONG or echoes message |
| `SET` | `SET key value [EX seconds] [PX ms]` | Set a key with optional TTL |
| `GET` | `GET key` | Get value by key |
| `DEL` | `DEL key [key ...]` | Delete one or more keys |
| `EXISTS` | `EXISTS key [key ...]` | Check if keys exist |
| `EXPIRE` | `EXPIRE key seconds` | Set TTL on existing key |
| `TTL` | `TTL key` | Get remaining TTL in seconds |
| `INCR` | `INCR key` | Increment integer value by 1 |
| `DECR` | `DECR key` | Decrement integer value by 1 |
| `INCRBY` | `INCRBY key delta` | Increment by arbitrary integer |

### List Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `LPUSH` | `LPUSH key value [value ...]` | Insert at head (left) |
| `RPUSH` | `RPUSH key value [value ...]` | Insert at tail (right) |
| `LPOP` | `LPOP key [count]` | Remove and return from head |
| `RPOP` | `RPOP key [count]` | Remove and return from tail |
| `LRANGE` | `LRANGE key start stop` | Get range of elements |
| `LLEN` | `LLEN key` | Get list length |
| `LINDEX` | `LINDEX key index` | Get element at index |

### Set Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `SADD` | `SADD key member [member ...]` | Add members to set |
| `SREM` | `SREM key member [member ...]` | Remove members from set |
| `SMEMBERS` | `SMEMBERS key` | Get all members |
| `SISMEMBER` | `SISMEMBER key member` | Check if member exists |
| `SCARD` | `SCARD key` | Get set cardinality (size) |
| `SUNION` | `SUNION key [key ...]` | Union of multiple sets |
| `SINTER` | `SINTER key [key ...]` | Intersection of multiple sets |

### Transaction Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `MULTI` | `MULTI` | Start transaction |
| `EXEC` | `EXEC` | Execute queued commands |
| `DISCARD` | `DISCARD` | Abort transaction |

---

## 🔬 Deep Dives

### 1. Network Layer

**Location:** `internal/netlayer/`

The server uses a **goroutine-per-connection** model:

```go
func (l *Listener) Serve(ctx context.Context, handler func(net.Conn)) error {
    for {
        conn, err := l.ln.Accept()
        // ...
        go handler(conn)  // Each client gets its own goroutine
    }
}
```

**Why goroutine-per-connection?**
- Simple mental model
- Go's goroutines are cheap (~2KB stack)
- The scheduler handles the rest

**Graceful shutdown** uses context cancellation:
```go
case <-ctx.Done():
    return nil  // Stop accepting new connections
```

---

### 2. RESP Protocol

**Location:** `internal/protocol/resp/`

RESP (REdis Serialization Protocol) is dead simple:

| Type | Prefix | Example |
|------|--------|---------|
| Simple String | `+` | `+OK\r\n` |
| Error | `-` | `-ERR unknown command\r\n` |
| Integer | `:` | `:1000\r\n` |
| Bulk String | `$` | `$5\r\nhello\r\n` |
| Array | `*` | `*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n` |

The reader is a **streaming parser**—it doesn't buffer the entire message:

```go
func (rd *Reader) ReadValue() (Value, error) {
    prefix, _ := rd.r.ReadByte()
    
    switch ValueType(prefix) {
    case SimpleString: return rd.readSimpleString()
    case Integer:      return rd.readInteger()
    case BulkString:   return rd.readBulkString()
    case Array:        return rd.readArray()  // Recursive!
    case Error:        return rd.readError()
    }
}
```

---

### 3. Command Dispatch

**Location:** `internal/commands/`

Commands are registered at startup:

```go
func RegisterAll() {
    commands.Register("PING", Ping)
    commands.Register("SET", Set)
    commands.Register("GET", Get)
    // ...
}
```

The dispatcher looks up handlers by name:

```go
func Dispatch(v resp.Value) resp.Value {
    cmd, _ := Parse(v)                    // Extract command name + args
    handler, ok := Get(cmd.Name)          // Lookup in registry
    if !ok {
        return resp.ErrorValue("ERR unknown command")
    }
    return handler(cmd.Args)              // Execute
}
```

**Per-client context** enables transactions:

```go
type ClientContext struct {
    InTxn   bool         // Inside MULTI?
    TxQueue []resp.Value // Queued commands
}
```

---

### 4. In-Memory Store

**Location:** `internal/engine/store/`

The core data structure:

```go
type Store struct {
    mu   sync.RWMutex
    data map[string]*Entry
}

type Entry struct {
    Type   ValueType  // String, List, Set, Hash
    Value  any        // The actual data
    Expiry time.Time  // Zero means no expiry
}
```

**Thread safety**: All operations acquire the mutex. Currently uses a single lock—sharding would improve performance under high concurrency.

---

### 5. Expiration (Lazy + Active)

Redis uses a **two-pronged approach** to expiration:

#### Lazy Expiration
When you access a key, we check if it's expired:

```go
func (s *Store) Get(key string) (*Entry, bool) {
    e := s.data[key]
    if e.IsExpired() {
        delete(s.data, key)  // Expired? Delete it now
        return nil, false
    }
    return e, true
}
```

**Problem**: Keys that are never accessed again **never get deleted**. Memory leak!

#### Active Expiration (Background Sweeper)
Every 100ms, we sample 20 random keys with TTL:

```go
func (s *Store) expireCycle() {
    for {
        expired := s.sampleAndExpire()
        
        // If <25% were expired, we're done
        if expired < 5 {
            return
        }
        // Otherwise, keep sweeping (too many dead keys)
    }
}
```

**Why random sampling?**
- Scanning all keys is O(n)—too slow
- Random sampling gives a statistical picture
- If many are expired, sweep again immediately

**Fisher-Yates shuffle** ensures fair random selection:

```go
for i := 0; i < sampleSize; i++ {
    j := i + rand.Intn(len(keys)-i)
    keys[i], keys[j] = keys[j], keys[i]
}
```

---

### 6. AOF Persistence

**Location:** `internal/persistence/`

**Append-Only File** logs every write command:

```
*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$5\r\nhello\r\n
*2\r\n$3\r\nDEL\r\n$5\r\nmykey\r\n
```

Commands are RESP-encoded—the same format used over the wire.

#### Async Write Pipeline

```go
type AOF struct {
    ch     chan []byte    // Buffer up to 1024 commands
    stopCh chan struct{}
}

func (a *AOF) Append(data []byte) {
    a.ch <- data  // Non-blocking send to background writer
}

func (a *AOF) Run() {
    go func() {
        for data := range a.ch {
            a.file.Write(data)
        }
    }()
}
```

#### Replay on Startup

```go
func (s *Server) Start() error {
    persistence.Replay("appendonly.aof", func(v resp.Value) {
        commands.Dispatch(v)  // Re-execute each command
    })
    // ...
}
```

#### Idempotent Logging

INCR is logged as SET to ensure replay safety:

```go
func Incr(args []string) resp.Value {
    result := incrBy(args[0], 1)
    // Log: SET counter 5 (not INCR counter)
    logCommand("SET", args[0], strconv.FormatInt(result.Int, 10))
    return result
}
```

Why? If we logged `INCR counter` and replayed it twice, we'd get the wrong value.

---

### 7. Transactions

**Location:** `internal/commands/dispatcher.go`

Transactions are dead simple—no rollback, just **delayed execution**:

```go
case "MULTI":
    ctx.InTxn = true
    ctx.TxQueue = nil
    return resp.SimpleValue("OK")

case "EXEC":
    results := []resp.Value{}
    for _, cmd := range ctx.TxQueue {
        results = append(results, Dispatch(cmd))
    }
    ctx.InTxn = false
    return resp.ArrayValue(results)
```

When inside a transaction, commands are **queued**, not executed:

```go
if ctx.InTxn {
    ctx.TxQueue = append(ctx.TxQueue, v)
    return resp.SimpleValue("QUEUED")
}
```

---

## 📁 Project Structure

```
go-redis/
├── cmd/
│   ├── server/           # Main server entry point
│   ├── testclient/       # Integration test client
│   ├── test_expiry/      # Expiration test
│   └── verify_replay/    # AOF replay verification
├── internal/
│   ├── commands/
│   │   ├── handlers/     # PING, SET, GET, etc.
│   │   ├── command.go    # Command parsing
│   │   ├── dispatcher.go # Routing + transactions
│   │   └── registry.go   # Handler registration
│   ├── engine/
│   │   └── store/        # In-memory data store
│   ├── netlayer/         # TCP server
│   ├── persistence/      # AOF logging + replay
│   ├── protocol/
│   │   └── resp/         # RESP reader/writer
│   └── server/           # Server orchestration
└── appendonly.aof        # Persistence file (generated)
```

---

## 🧪 Running Tests

```bash
# Start the server
go run ./cmd/server

# Run integration tests (in another terminal)
go run ./cmd/testclient

# Test expiration
go run ./cmd/test_expiry

# Verify AOF replay (restart server, then)
go run ./cmd/verify_replay
```

---

## 📊 Benchmarks

### Performance Comparison: go-redis vs Real Redis

We benchmarked go-redis against the official Redis server to see how our learning implementation stacks up against the battle-tested original.

**Test Configuration:**
- 50 parallel clients
- 10,000 total requests per command
- 3-byte payload size

#### Results Summary

| Command | go-redis (ops/sec) | Real Redis (ops/sec) | go-redis % of Redis |
|---------|-------------------|---------------------|---------------------|
| **PING** | 86,356 | N/A | — |
| **SET** | 54,343 | 111,111 | 49% |
| **GET** | 86,178 | 105,263 | 82% |
| **INCR** | 51,908 | 102,040 | 51% |
| **LPUSH** | 5,886 | 80,000 | 7% |
| **SADD** | 71,201 | 111,111 | 64% |

#### Detailed go-redis Results

```
╔═══════════════════════════════════════════════════════════╗
║              go-redis Benchmark Tool                       ║
╚═══════════════════════════════════════════════════════════╝

Server: localhost:6379
Clients: 50, Requests: 10000, Data size: 3 bytes

╔═══════════════════════════════════════════════════════════╗
║                      SUMMARY                               ║
╠═══════════════════════════════════════════════════════════╣
║ Command    │      Ops/sec │  Avg Latency │   Total Time   ║
╠═══════════════════════════════════════════════════════════╣
║ PING       │      86356/s │     458.00µs │       0.12s    ║
║ SET        │      54343/s │     860.00µs │       0.18s    ║
║ GET        │      86178/s │     528.00µs │       0.12s    ║
║ INCR       │      51908/s │     860.00µs │       0.19s    ║
║ LPUSH      │       5886/s │    8304.00µs │       1.70s    ║
║ SADD       │      71201/s │     626.00µs │       0.14s    ║
╚═══════════════════════════════════════════════════════════╝
```

#### Real Redis Results (via redis-benchmark)

```
SET:    111,111 requests/sec   avg_latency: 0.283ms   p99: 1.119ms
GET:    105,263 requests/sec   avg_latency: 0.290ms   p99: 1.199ms
INCR:   102,040 requests/sec   avg_latency: 0.296ms   p99: 2.439ms
LPUSH:   80,000 requests/sec   avg_latency: 0.364ms   p99: 2.239ms
SADD:   111,111 requests/sec   avg_latency: 0.283ms   p99: 0.999ms
```

### Analysis

**What's Working Well:**
- **GET operations** achieve 82% of Redis performance—great for a learning project!
- **SADD** shows solid 64% performance with set operations
- **PING** at 86K ops/sec demonstrates efficient network handling

**Areas for Improvement:**
- **LPUSH** is significantly slower (7%)—likely due to list implementation or locking overhead
- **SET/INCR** operations are around 50%—AOF persistence adds overhead compared to Redis's in-memory-only benchmark

**Why the Difference?**
1. **Single global lock** vs Redis's more sophisticated locking strategies
2. **AOF persistence enabled** during benchmarks (Redis benchmark runs with `appendonly: no`)
3. **Learning-focused code** prioritizing clarity over raw performance

### How to Run Benchmarks

#### 1. Benchmark Your go-redis

```bash
# Start the server
go run ./cmd/server

# In another terminal, run the benchmark
go run ./cmd/benchmark -c 50 -n 10000

# Available options:
#   -h localhost   Server hostname
#   -p 6379        Server port
#   -c 50          Number of parallel clients
#   -n 10000       Total requests
#   -d 3           Data size in bytes
#   -t all         Test type: set, get, incr, lpush, sadd, all

# Examples:
go run ./cmd/benchmark -c 100 -n 100000        # Heavy load test
go run ./cmd/benchmark -c 50 -n 50000 -t set   # Just SET operations
```

#### 2. Compare with Real Redis

**Option A: Using Docker (Recommended)**

```bash
# Start Redis in Docker (port 6380 to avoid conflict)
docker run -d --name redis-test -p 6380:6379 redis:latest

# Run our benchmark tool against real Redis
go run ./cmd/benchmark -c 50 -n 10000 -p 6380

# Run Redis's native benchmark (runs inside container for accurate results)
docker exec redis-test redis-benchmark -c 50 -n 10000 -t set,get,incr,lpush,sadd

# Cleanup
docker stop redis-test && docker rm redis-test
```

**Option B: Local Redis Installation**

```bash
# Start real Redis on port 6380
redis-server --port 6380

# Benchmark real Redis
redis-benchmark -h localhost -p 6380 -c 50 -n 10000 -t set,get,incr,lpush,sadd

# Benchmark go-redis (same parameters)
go run ./cmd/benchmark -c 50 -n 10000
```

> **Note:** When benchmarking Redis via Docker on Windows/Mac, network overhead can significantly impact results. For accurate comparisons, run redis-benchmark inside the container or use native Redis installation.

---

## 🛣 What's Next

| Feature | Status |
|---------|--------|
| String commands | ✅ Done |
| List commands (LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN, LINDEX) | ✅ Done |
| Set commands (SADD, SREM, SMEMBERS, SISMEMBER, SCARD, SUNION, SINTER) | ✅ Done |
| TTL / Expiration | ✅ Done |
| Active expiration sweeper | ✅ Done |
| AOF persistence | ✅ Done |
| Transactions (MULTI/EXEC) | ✅ Done |
| Hash commands (HSET, HGET, etc.) | 🔜 Planned |
| Pub/Sub | 🔜 Planned |
| WATCH for optimistic locking | 🔜 Planned |
| AOF rewrite/compaction | 🔜 Planned |
| Sharded locks for better concurrency | 🔜 Planned |

---

## 📚 Learning Resources

- [Redis Internals](https://redis.io/docs/reference/internals/)
- [RESP Protocol Specification](https://redis.io/docs/reference/protocol-spec/)
- [Redis Persistence](https://redis.io/docs/management/persistence/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)

---

## 🤝 Contributing

This is a learning project! Feel free to:
- Add new commands
- Improve concurrency (sharded locks)
- Add RDB snapshots
- Implement Pub/Sub
- Write unit tests

---

## 📄 License

MIT

---

**Built with ❤️ and a lot of `fmt.Println` debugging.**
