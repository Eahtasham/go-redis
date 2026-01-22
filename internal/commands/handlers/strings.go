package handlers

import (
	"strconv"
	"time"

	"github.com/Eahtasham/go-redis/internal/engine/store"
	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

// Global store instance - will be initialized at server startup
var Store *store.Store

// InitStore sets the global store instance
func InitStore(s *store.Store) {
	Store = s
}

// Ping handles the PING command
func Ping(args []string) resp.Value {
	if len(args) > 0 {
		return resp.BulkValue(args[0])
	}
	return resp.SimpleValue("PONG")
}

// Set handles the SET command
// SET key value [EX seconds] [PX milliseconds]
func Set(args []string) resp.Value {
	if len(args) < 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'set' command")
	}

	key := args[0]
	value := args[1]

	Store.Set(key, store.StringType, value)

	// Handle optional EX/PX arguments
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			if i+1 >= len(args) {
				return resp.ErrorValue("ERR syntax error")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorValue("ERR value is not an integer or out of range")
			}
			Store.SetExpiry(key, time.Duration(seconds)*time.Second)
			i++
		case "PX", "px":
			if i+1 >= len(args) {
				return resp.ErrorValue("ERR syntax error")
			}
			ms, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorValue("ERR value is not an integer or out of range")
			}
			Store.SetExpiry(key, time.Duration(ms)*time.Millisecond)
			i++
		}
	}

	return resp.SimpleValue("OK")
}

// Get handles the GET command
func Get(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'get' command")
	}

	key := args[0]
	entry, ok := Store.Get(key)
	if !ok {
		return resp.Value{Type: resp.BulkString, Str: ""} // nil bulk string
	}

	if entry.Type != store.StringType {
		return resp.ErrorValue("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return resp.BulkValue(entry.Value.(string))
}

// Del handles the DEL command
func Del(args []string) resp.Value {
	if len(args) == 0 {
		return resp.ErrorValue("ERR wrong number of arguments for 'del' command")
	}

	count := int64(0)
	for _, key := range args {
		if Store.Delete(key) {
			count++
		}
	}

	return resp.IntValue(count)
}

// Exists handles the EXISTS command
func Exists(args []string) resp.Value {
	if len(args) == 0 {
		return resp.ErrorValue("ERR wrong number of arguments for 'exists' command")
	}

	count := int64(0)
	for _, key := range args {
		if _, ok := Store.Get(key); ok {
			count++
		}
	}

	return resp.IntValue(count)
}

// Expire handles the EXPIRE command
func Expire(args []string) resp.Value {
	if len(args) != 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'expire' command")
	}

	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}

	if Store.SetExpiry(key, time.Duration(seconds)*time.Second) {
		return resp.IntValue(1)
	}
	return resp.IntValue(0)
}

// TTL handles the TTL command
func TTL(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'ttl' command")
	}

	key := args[0]
	entry, ok := Store.Get(key)
	if !ok {
		return resp.IntValue(-2) // key does not exist
	}

	if entry.Expiry.IsZero() {
		return resp.IntValue(-1) // key exists but has no expiry
	}

	ttl := time.Until(entry.Expiry).Seconds()
	if ttl < 0 {
		return resp.IntValue(-2)
	}

	return resp.IntValue(int64(ttl))
}

// Incr handles the INCR command
func Incr(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'incr' command")
	}
	return incrBy(args[0], 1)
}

// Decr handles the DECR command
func Decr(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'decr' command")
	}
	return incrBy(args[0], -1)
}

// IncrBy handles the INCRBY command
func IncrBy(args []string) resp.Value {
	if len(args) != 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'incrby' command")
	}

	delta, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}

	return incrBy(args[0], delta)
}

// incrBy is the internal helper for INCR/DECR/INCRBY
func incrBy(key string, delta int64) resp.Value {
	entry, ok := Store.Get(key)

	var current int64 = 0
	if ok {
		if entry.Type != store.StringType {
			return resp.ErrorValue("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		val, err := strconv.ParseInt(entry.Value.(string), 10, 64)
		if err != nil {
			return resp.ErrorValue("ERR value is not an integer or out of range")
		}
		current = val
	}

	newVal := current + delta
	Store.Set(key, store.StringType, strconv.FormatInt(newVal, 10))

	return resp.IntValue(newVal)
}
