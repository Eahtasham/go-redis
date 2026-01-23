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

	// List commands
	commands.Register("LPUSH", LPush)
	commands.Register("RPUSH", RPush)
	commands.Register("LPOP", LPop)
	commands.Register("RPOP", RPop)
	commands.Register("LRANGE", LRange)
	commands.Register("LLEN", LLen)
	commands.Register("LINDEX", LIndex)

	// Set commands
	commands.Register("SADD", SAdd)
	commands.Register("SREM", SRem)
	commands.Register("SMEMBERS", SMembers)
	commands.Register("SISMEMBER", SIsMember)
	commands.Register("SCARD", SCard)
	commands.Register("SUNION", SUnion)
	commands.Register("SINTER", SInter)
}
