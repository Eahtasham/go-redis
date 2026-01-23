package persistence

import "os"

type AOF struct {
	file   *os.File
	ch     chan []byte
	stopCh chan struct{}
	doneCh chan struct{} // signals when background writer has finished
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
		doneCh: make(chan struct{}),
	}, nil
}

func (a *AOF) Run() {
	go func() {
		defer close(a.doneCh)
		for {
			select {
			case data := <-a.ch:
				a.file.Write(data)
			case <-a.stopCh:
				// Drain remaining commands before closing
				for {
					select {
					case data := <-a.ch:
						a.file.Write(data)
					default:
						a.file.Sync()
						a.file.Close()
						return
					}
				}
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

// Stop signals the background writer to stop and waits for completion
func (a *AOF) Stop() {
	close(a.stopCh)
	<-a.doneCh // wait for background writer to finish
}
