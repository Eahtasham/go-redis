package handlers

import (
	"github.com/Eahtasham/go-redis/internal/commands"
)

// RegisterAll registers all command handlers
func RegisterAll() {
	// String commands
	commands.Register("PING", Ping)
	commands.Register("SET", Set)
	commands.Register("GET", Get)
	commands.Register("DEL", Del)
	commands.Register("EXISTS", Exists)
	commands.Register("EXPIRE", Expire)
	commands.Register("TTL", TTL)
	commands.Register("INCR", Incr)
	commands.Register("DECR", Decr)
	commands.Register("INCRBY", IncrBy)

	// TODO: Add more commands as they are implemented
	// List commands: LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN
	// Set commands: SADD, SREM, SMEMBERS, SISMEMBER
	// Hash commands: HSET, HGET, HDEL, HGETALL
}
