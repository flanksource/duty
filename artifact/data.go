package artifact

import (
	"fmt"
	"io"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/gabriel-vasile/mimetype"
)

type Data struct {
	Content       io.ReadCloser
	ContentLength int64
	Checksum      string
	ContentType   string
	Filename      string
}

func (d Data) Pretty() api.Text {
	s := clicky.Text(d.Filename, "font-bold")
	if d.ContentType != "" {
		s = s.AddText(" "+d.ContentType, "text-gray-500")
	}
	if d.ContentLength > 0 {
		s = s.AddText(fmt.Sprintf(" (%s)", formatBytes(d.ContentLength)), "text-gray-400")
	}
	if d.Checksum != "" {
		short := d.Checksum
		if len(short) > 8 {
			short = short[:8]
		}
		s = s.AddText(" sha:"+short, "text-yellow-500")
	}
	return s
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

const maxBytesForMimeDetection = 512 * 1024

type mimeWriter struct {
	buffer []byte
	Max    int
}

func (t *mimeWriter) Write(bb []byte) (n int, err error) {
	if len(t.buffer) < t.Max {
		rem := t.Max - len(t.buffer)
		if rem > len(bb) {
			rem = len(bb)
		}
		t.buffer = append(t.buffer, bb[:rem]...)
	}
	return len(bb), nil
}

func (t *mimeWriter) Detect() *mimetype.MIME {
	return mimetype.Detect(t.buffer)
}
