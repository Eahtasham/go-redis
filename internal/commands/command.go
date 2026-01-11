package commands

import (
	"errors"
	"strings"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

type Command struct {
	Name string
	Args []string
}

// Here we will only check symantical correctness, syntax errors will be handled later
func Parse(v resp.Value) (Command, error) {
	if v.Type != resp.Array || len(v.Array) == 0 {
		return Command{}, errors.New("ERR invalid command")
	}

	name := strings.ToUpper(v.Array[0].Str)
	args := make([]string, 0, len(v.Array)-1)

	for _, arg := range v.Array[1:] {
		args = append(args, arg.Str)
	}

	return Command{
		Name: name,
		Args: args,
	}, nil
}
