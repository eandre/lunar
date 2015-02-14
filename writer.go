package lunar

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

const indent = "\t"

var newline = []byte{'\n'}
var strNewline = string(newline)

type WriteError struct {
	Err error
}

func (e WriteError) Error() string {
	return e.Err.Error()
}

type Writer struct {
	w         io.Writer
	level     int
	prefix    []byte
	isNewline bool
}

func (w *Writer) Write(p []byte) (int, error) {
	// TODO(eandre) We should probably split on each newline to re-indent
	// Write line prefix if we're on a new line
	if w.isNewline {
		_, err := w.w.Write(w.prefix)
		w.err(err)
	}

	_, err := w.w.Write(p)
	w.err(err)
	w.isNewline = bytes.HasSuffix(p, newline)
	return len(p), nil
}

func (w *Writer) WriteByte(p byte) {
	w.Write([]byte{p})
}

func (w *Writer) WriteBytes(p []byte) {
	w.Write(p)
}

func (w *Writer) WriteString(s string) {
	w.Write([]byte(s))
}

func (w *Writer) WriteLine(line string) {
	w.WriteString(line)
	if !strings.HasSuffix(line, strNewline) {
		w.WriteNewline()
	}
}

func (w *Writer) WriteNewline() {
	w.Write(newline)
}

func (w *Writer) WriteLinef(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	w.WriteLine(line)
}

func (w *Writer) Indent() {
	w.level += 1
	w.prefix = []byte(strings.Repeat(indent, w.level))
}

func (w *Writer) Dedent() {
	w.level -= 1
	if w.level < 0 {
		panic("lunar: Writer.Dedent called when at indentation level 0")
	}
	w.prefix = []byte(strings.Repeat(indent, w.level))
}

func (w *Writer) err(e error) {
	if e != nil {
		panic(WriteError{Err: e})
	}
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:         w,
		isNewline: true,
	}
}
