package resp

type ValueType byte

const (
	SimpleString ValueType = '+'
	Error        ValueType = '-'
	Integer      ValueType = ':'
	BulkString   ValueType = '$'
	Array        ValueType = '*'
)

type Value struct {
	Type  ValueType
	Str   string
	Int   int64
	Array []Value
}
