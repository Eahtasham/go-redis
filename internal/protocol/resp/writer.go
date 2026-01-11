package resp

import (
	"bufio"
	"fmt"
	"io"
)

type Writer struct {
	w *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: bufio.NewWriter(w),
	}
}

func (wr *Writer) WriteValue(v Value) error {
	switch v.Type {
	case SimpleString:
		if _, err := fmt.Fprintf(wr.w, "+%s\r\n", v.Str); err != nil {
			return err
		}
		return wr.w.Flush()
	case Integer:
		if _, err := fmt.Fprintf(wr.w, ":%d\r\n", v.Int); err != nil {
			return err
		}
		return wr.w.Flush()
	case BulkString:
		if v.Str == "" {
			if _, err := wr.w.WriteString("$-1\r\n"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(wr.w, "$%d\r\n%s\r\n", len(v.Str), v.Str); err != nil {
				return err
			}
		}
		return wr.w.Flush()
	case Array:
		_, err := fmt.Fprintf(wr.w, "*%d\r\n", len(v.Array))
		if err != nil {
			return err
		}
		for _, el := range v.Array {
			if err = wr.WriteValue(el); err != nil {
				return err
			}
		}
	}
	return wr.w.Flush()
}
