package lunar_test

import (
	"bytes"
	"fmt"
	"github.com/eandre/lunar"
	"testing"
)

func TestWriteLine(t *testing.T) {
	buf := &bytes.Buffer{}
	w := lunar.NewWriter(buf)

	// Write one line
	w.WriteLine("foo")
	want := "foo\n"
	if s := buf.String(); s != want {
		t.Fatalf("Got output %q, want %q", s, want)
	}

	// Indent and write two more
	w.Indent()
	w.Indent()
	w.WriteLine("bar boo")
	w.WriteLine("hooray")
	want += "\t\tbar boo\n\t\thooray\n"
	if s := buf.String(); s != want {
		t.Fatalf("Got output %q, want %q", s, want)
	}

	// Dedent and write another
	w.Dedent()
	w.WriteLine("dedented")
	want += "\tdedented\n"
	if s := buf.String(); s != want {
		t.Fatalf("Got output %q, want %q", s, want)
	}

	// Dedent and write another
	w.Dedent()
	w.WriteLine("back to square one")
	want += "back to square one\n"
	if s := buf.String(); s != want {
		t.Fatalf("Got output %q, want %q", s, want)
	}
}

func TestWriteLinef(t *testing.T) {
	buf := &bytes.Buffer{}
	w := lunar.NewWriter(buf)

	want := fmt.Sprintf("\tfoo %s %d hi\n", "foo", 5)
	w.Indent()
	w.WriteLinef("foo %s %d hi", "foo", 5)
	if s := buf.String(); s != want {
		t.Fatalf("Got output %q, want %q", s, want)
	}
}
