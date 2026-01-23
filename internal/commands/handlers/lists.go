package handlers

import (
	"strconv"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

// LPUSH key value [value ...]
// Insert values at the head (left) of the list
func LPush(args []string) resp.Value {
	if len(args) < 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'lpush' command")
	}

	key := args[0]
	values := args[1:]

	length, err := Store.LPush(key, values)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	// Log to AOF
	logCommand("LPUSH", args...)

	return resp.IntValue(length)
}

// RPUSH key value [value ...]
// Insert values at the tail (right) of the list
func RPush(args []string) resp.Value {
	if len(args) < 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'rpush' command")
	}

	key := args[0]
	values := args[1:]

	length, err := Store.RPush(key, values)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	// Log to AOF
	logCommand("RPUSH", args...)

	return resp.IntValue(length)
}

// LPOP key [count]
// Remove and return element(s) from the head (left) of the list
func LPop(args []string) resp.Value {
	if len(args) < 1 || len(args) > 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'lpop' command")
	}

	key := args[0]
	count := 1
	if len(args) == 2 {
		var err error
		count, err = strconv.Atoi(args[1])
		if err != nil || count < 0 {
			return resp.ErrorValue("ERR value is not an integer or out of range")
		}
	}

	popped, err := Store.LPop(key, count)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	if popped == nil {
		return resp.Value{Type: resp.BulkString, Str: ""} // nil
	}

	// Log state for AOF (get remaining list or DEL if empty)
	if remaining, exists := Store.GetListCopy(key); exists {
		logCommand("DEL", key)
		rpushArgs := append([]string{key}, remaining...)
		logCommand("RPUSH", rpushArgs...)
	} else {
		logCommand("DEL", key)
	}

	// Return single element or array based on count
	if len(args) == 1 {
		return resp.BulkValue(popped[0])
	}

	result := make([]resp.Value, len(popped))
	for i, v := range popped {
		result[i] = resp.BulkValue(v)
	}
	return resp.ArrayValue(result)
}

// RPOP key [count]
// Remove and return element(s) from the tail (right) of the list
func RPop(args []string) resp.Value {
	if len(args) < 1 || len(args) > 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'rpop' command")
	}

	key := args[0]
	count := 1
	if len(args) == 2 {
		var err error
		count, err = strconv.Atoi(args[1])
		if err != nil || count < 0 {
			return resp.ErrorValue("ERR value is not an integer or out of range")
		}
	}

	popped, err := Store.RPop(key, count)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	if popped == nil {
		return resp.Value{Type: resp.BulkString, Str: ""} // nil
	}

	// Log state for AOF (get remaining list or DEL if empty)
	if remaining, exists := Store.GetListCopy(key); exists {
		logCommand("DEL", key)
		rpushArgs := append([]string{key}, remaining...)
		logCommand("RPUSH", rpushArgs...)
	} else {
		logCommand("DEL", key)
	}

	// Return single element or array based on count
	if len(args) == 1 {
		return resp.BulkValue(popped[0])
	}

	result := make([]resp.Value, len(popped))
	for i, v := range popped {
		result[i] = resp.BulkValue(v)
	}
	return resp.ArrayValue(result)
}

// LRANGE key start stop
// Get a range of elements from the list
func LRange(args []string) resp.Value {
	if len(args) != 3 {
		return resp.ErrorValue("ERR wrong number of arguments for 'lrange' command")
	}

	key := args[0]
	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}

	elements, err := Store.LRange(key, start, stop)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	result := make([]resp.Value, len(elements))
	for i, v := range elements {
		result[i] = resp.BulkValue(v)
	}

	return resp.ArrayValue(result)
}

// LLEN key
// Get the length of the list
func LLen(args []string) resp.Value {
	if len(args) != 1 {
		return resp.ErrorValue("ERR wrong number of arguments for 'llen' command")
	}

	key := args[0]
	length, err := Store.LLen(key)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	return resp.IntValue(length)
}

// LINDEX key index
// Get element at index
func LIndex(args []string) resp.Value {
	if len(args) != 2 {
		return resp.ErrorValue("ERR wrong number of arguments for 'lindex' command")
	}

	key := args[0]
	index, err := strconv.Atoi(args[1])
	if err != nil {
		return resp.ErrorValue("ERR value is not an integer or out of range")
	}

	value, exists, err := Store.LIndex(key, index)
	if err != nil {
		return resp.ErrorValue(err.Error())
	}

	if !exists {
		return resp.Value{Type: resp.BulkString, Str: ""} // nil
	}

	return resp.BulkValue(value)
}
