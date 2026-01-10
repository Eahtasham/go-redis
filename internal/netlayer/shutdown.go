package netlayer

func (l *Listener) Close() error {
	err := l.ln.Close()
	l.wg.Wait()
	return err
}
