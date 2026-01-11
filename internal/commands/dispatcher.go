package commands

import "github.com/Eahtasham/go-redis/internal/protocol/resp"

func Dispatch(v resp.Value) resp.Value {
	cmd, err := Parse(v)
	if err != nil {
		return resp.Value{
			Type: resp.Error,
			Str:  err.Error(),
		}
	}

	handler, ok := Get(cmd.Name)
	if !ok {
		return resp.Value{
			Type: resp.Error,
			Str:  "ERR unknown command '" + cmd.Name + "'",
		}
	}

	return handler(cmd.Args)

}
