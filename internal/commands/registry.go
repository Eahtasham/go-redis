package commands

import (
	"strings"

	"github.com/Eahtasham/go-redis/internal/protocol/resp"
)

type Handler func(args []string) resp.Value

var handlers = map[string]Handler{}

func Register(name string, h Handler) {
	handlers[strings.ToUpper(name)] = h
}

func Get(name string) (Handler, bool) {
	h, ok := handlers[strings.ToUpper(name)]
	return h, ok
}
