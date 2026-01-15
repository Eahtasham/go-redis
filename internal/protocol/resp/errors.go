package resp

// ErrorValue creates an error response Value
func ErrorValue(msg string) Value {
	return Value{Type: Error, Str: msg}
}

// SimpleValue creates a simple string response Value
func SimpleValue(msg string) Value {
	return Value{Type: SimpleString, Str: msg}
}

// IntValue creates an integer response Value
func IntValue(n int64) Value {
	return Value{Type: Integer, Int: n}
}

// BulkValue creates a bulk string response Value
func BulkValue(s string) Value {
	return Value{Type: BulkString, Str: s}
}

// ArrayValue creates an array response Value
func ArrayValue(arr []Value) Value {
	return Value{Type: Array, Array: arr}
}
