package server

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

const LostThreshold = 5 * time.Minute

type Argument struct {
	Ticket string

	Name string
	Host string
	Port int
}

type Server interface {
	Listen(addr string) (err error)
	AddPassages(passages []Passage) (err error)
	RemovePassages(passages []Passage, alsoManager bool) (err error)
	SyncPassages(passages []Passage) (err error)
	Passages() (passages []Passage)
	io.Closer
}

type Creator func(sweetLisaHost, chatIdentifier string, arg Argument) (Server, error)

var Mapper = make(map[string]Creator)

func Register(name string, c Creator) {
	Mapper[name] = c
}

func NewServer(protocol string, sweetLisaHost, chatIdentifier string, arg Argument) (Server, error) {
	creator, ok := Mapper[protocol]
	if !ok {
		return nil, fmt.Errorf("no server creator registered for %v", strconv.Quote(protocol))
	}
	return creator(sweetLisaHost, chatIdentifier, arg)
}
