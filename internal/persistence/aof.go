package persistence

import "os"

type AOF struct {
	file   *os.File
	ch     chan []byte
	stopCh chan struct{}
}

func NewAOF(path string) (*AOF, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return nil, err
	}

	return &AOF{
		file:   f,
		ch:     make(chan []byte, 1024),
		stopCh: make(chan struct{}),
	}, nil
}

func (a *AOF) Run() {
	go func() {
		for {
			select {
			case data := <-a.ch:
				a.file.Write(data)
			case <-a.stopCh:
				a.file.Sync()
				a.file.Close()
				return

			}
		}
	}()
}

func (a *AOF) Append(data []byte) {
	select {
	case a.ch <- data:
	default:
		// drop or block later; for now, drop is acceptable
	}
}
