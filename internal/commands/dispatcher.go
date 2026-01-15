package commands

import "github.com/Eahtasham/go-redis/internal/protocol/resp"

// ClientContext holds per-client state for features like transactions
type ClientContext struct {
	InTxn   bool         // true when inside a MULTI transaction
	TxQueue []resp.Value // queued commands during a transaction
}

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

func DispatchWithContext(v resp.Value, ctx *ClientContext) resp.Value {
	cmd, err := Parse(v)
	if err != nil {
		return resp.ErrorValue("ERR invalid command")
	}

	switch cmd.Name {
	case "MULTI":
		ctx.InTxn = true
		ctx.TxQueue = nil
		return resp.SimpleValue("OK")

	case "DISCARD":
		ctx.InTxn = false
		ctx.TxQueue = nil
		return resp.SimpleValue("OK")

	case "EXEC":
		if !ctx.InTxn {
			return resp.ErrorValue("ERR EXEC without MULTI")
		}
		return execTransaction(ctx)
	}

	// normal command
	if ctx.InTxn {
		ctx.TxQueue = append(ctx.TxQueue, v)
		return resp.SimpleValue("QUEUED")
	}

	return Dispatch(v)
}

func execTransaction(ctx *ClientContext) resp.Value {
	ctx.InTxn = false

	results := make([]resp.Value, 0, len(ctx.TxQueue))

	for _, v := range ctx.TxQueue {
		res := Dispatch(v)
		results = append(results, res)
	}

	ctx.TxQueue = nil

	return resp.Value{
		Type:  resp.Array,
		Array: results,
	}
}
