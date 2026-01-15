package persistence

import (
	"bytes"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

func EncodeCommand(cmd string, args []string) []byte {
	var buf bytes.Buffer
	w := resp.NewWriter(&buf)

	vals := make([]resp.Value, 0, len(args)+1)
	vals = append(vals, resp.Value{Type: resp.BulkString, Str: cmd})

	for _, a := range args {
		vals = append(vals, resp.Value{Type: resp.BulkString, Str: a})
	}

	w.WriteValue(resp.Value{
		Type:  resp.Array,
		Array: vals,
	})

	return buf.Bytes()
}
