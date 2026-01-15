package persistence

import (
	"os"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

func Replay(path string, dispatch func(resp.Value)) error {
	f, err := os.Open(path)
	if err != nil {
		return nil // no AOF yet
	}
	defer f.Close()

	r := resp.NewReader(f)

	for {
		v, err := r.ReadValue()
		if err != nil {
			break
		}
		dispatch(v)
	}

	return nil
}
