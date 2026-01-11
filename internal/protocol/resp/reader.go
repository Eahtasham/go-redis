package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Reader struct {
	r *bufio.Reader
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(rd),
	}
}

func (rd *Reader) ReadValue() (Value, error) {
	prefix, err := rd.r.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch ValueType(prefix) {
	case SimpleString:
		return rd.readSimpleString()
	case Integer:
		return rd.readInteger()
	case BulkString:
		return rd.readBulkString()
	case Array:
		return rd.readArray()
	case Error:
		return rd.readError()
	default:
		return Value{}, fmt.Errorf("unknown RESP type: %q", prefix)
	}
}

func (rd *Reader) readLine() (string, error) {
	line, err := rd.r.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(line, "\r\n"), nil
}

func (rd *Reader) readSimpleString() (Value, error) {
	line, err := rd.readLine()
	if err != nil {
		return Value{}, err
	}

	return Value{
		Type: SimpleString,
		Str:  line,
	}, nil
}

func (rd *Reader) readInteger() (Value, error) {
	line, err := rd.readLine()
	if err != nil {
		return Value{}, err
	}

	n, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return Value{}, err
	}

	return Value{
		Type: Integer,
		Int:  n,
	}, nil
}

func (rd *Reader) readBulkString() (Value, error) {
	line, err := rd.readLine()
	if err != nil {
		return Value{}, err
	}

	size, _ := strconv.Atoi(line)

	if size == -1 {
		return Value{Type: BulkString}, nil
	}

	buf := make([]byte, size+2)
	if _, err := io.ReadFull(rd.r, buf); err != nil {
		return Value{}, err
	}

	return Value{
		Type: BulkString,
		Str:  string(buf[:size]),
	}, nil

}

func (rd *Reader) readArray() (Value, error) {
	line, err := rd.readLine()
	if err != nil {
		return Value{}, err
	}

	count, err := strconv.Atoi(line)
	if err != nil {
		return Value{}, err
	}

	arr := make([]Value, 0, count)
	for i := 0; i < count; i++ {
		v, err := rd.ReadValue()
		if err != nil {
			return Value{}, err
		}
		arr = append(arr, v)
	}

	return Value{
		Type:  Array,
		Array: arr,
	}, nil
}

func (rd *Reader) readError() (Value, error) {
	line, err := rd.readLine()
	if err != nil {
		return Value{}, err
	}

	return Value{
		Type: Error,
		Str:  line,
	}, nil
}
