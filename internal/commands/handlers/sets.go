package handlers

import (
	"github.com/Eahtasham/go-redis/internal/engine/store"
	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

// SADD key member [member ...]
// Add members to a set
func SAdd(args []string) resp.Value {
	if len(args) < 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'sadd' command")
	}

	key := args[0]
	members := args[1:]

	added, err := Store.SAdd(key, members)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	// Log to AOF
	logCommand("SADD", args...)

	return resp.IntValue(added)
}

// SREM key member [member ...]
// Remove members from a set
func SRem(args []string) resp.Value {
	if len(args) < 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'srem' command")
	}

	key := args[0]
	members := args[1:]

	removed, exists := Store.SRem(key, members)
	if !exists {
		return resp.IntValue(0)
	}

	if removed > 0 {
		logCommand("SREM", args...)
	}

	return resp.IntValue(removed)
}

// SMEMBERS key
// Get all members of a set
func SMembers(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'smembers' command")
	}

	key := args[0]
	members, err := Store.SMembers(key)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	result := make([]resp.Value, len(members))
	for i, member := range members {
		result[i] = resp.BulkValue(member)
	}

	return resp.ArrayValue(result)
}

// SISMEMBER key member
// Check if member exists in set
func SIsMember(args []string) resp.Value {
	if len(args) != 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'sismember' command")
	}

	key := args[0]
	member := args[1]

	exists, err := Store.SIsMember(key, member)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	if exists {
		return resp.IntValue(1)
	}
	return resp.IntValue(0)
}

// SCARD key
// Get the number of members in a set
func SCard(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'scard' command")
	}

	key := args[0]
	count, err := Store.SCard(key)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	return resp.IntValue(count)
}

// SUNION key [key ...]
// Return the union of multiple sets
func SUnion(args []string) resp.Value {
	if len(args) == 0 {
		return resp.ErrorValue("ERR wrong number of arguments for 'sunion' command")
	}

	result := make(map[string]struct{})

	for _, key := range args {
		members, err := Store.SMembers(key)
		if err != nil {
			if err == store.ErrWrongType {
				return resp.ErrorValue(err.Error())
			}
			continue
		}

		for _, member := range members {
			result[member] = struct{}{}
		}
	}

	arr := make([]resp.Value, 0, len(result))
	for member := range result {
		arr = append(arr, resp.BulkValue(member))
	}

	return resp.ArrayValue(arr)
}

// SINTER key [key ...]
// Return the intersection of multiple sets
func SInter(args []string) resp.Value {
	if len(args) == 0 {
		return resp.ErrorValue("ERR wrong number of arguments for 'sinter' command")
	}

	// Get the first set
	members, err := Store.SMembers(args[0])
	if err != nil {
		if err == store.ErrWrongType {
			return resp.ErrorValue(err.Error())
		}
		return resp.ArrayValue([]resp.Value{})
	}

	result := make(map[string]struct{})
	for _, member := range members {
		result[member] = struct{}{}
	}

	// Intersect with remaining sets
	for _, key := range args[1:] {
		members, err := Store.SMembers(key)
		if err != nil {
			if err == store.ErrWrongType {
				return resp.ErrorValue(err.Error())
			}
			return resp.ArrayValue([]resp.Value{}) // Empty intersection
		}

		setMembers := make(map[string]struct{})
		for _, m := range members {
			setMembers[m] = struct{}{}
		}

		// Keep only members that exist in both
		for member := range result {
			if _, exists := setMembers[member]; !exists {
				delete(result, member)
			}
		}
	}

	arr := make([]resp.Value, 0, len(result))
	for member := range result {
		arr = append(arr, resp.BulkValue(member))
	}

	return resp.ArrayValue(arr)
}
