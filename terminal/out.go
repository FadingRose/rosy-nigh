package terminal

import "io"

type transcriptWriter struct {
	w io.Writer
}

func (t *transcriptWriter) Echo(s string) {
	t.w.Write([]byte(s))
}
