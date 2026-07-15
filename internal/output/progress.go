package output

import (
	"fmt"
	"io"
	"sync/atomic"
)

// progressReader wraps a reader and reports bytes read to w as a single
// rewritten stderr line. It is a no-op display when total is unknown (<= 0).
type progressReader struct {
	src   io.Reader
	w     io.Writer
	total int64
	read  atomic.Int64
	label string
}

// NewProgressReader returns r wrapped so that read progress is rendered to w.
// When total <= 0 the wrapper still counts but prints nothing.
func NewProgressReader(r io.Reader, total int64, w io.Writer, label string) io.Reader {
	return &progressReader{src: r, w: w, total: total, label: label}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.src.Read(b)
	if n > 0 && p.total > 0 {
		done := p.read.Add(int64(n))
		pct := done * 100 / p.total
		fmt.Fprintf(p.w, "\r%s %d%% (%d/%d bytes)", p.label, pct, done, p.total)
		if err == io.EOF {
			fmt.Fprintln(p.w)
		}
	}
	return n, err
}
