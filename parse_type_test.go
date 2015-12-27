package lunar

import (
	"testing"
)

func TestStructType(t *testing.T) {
	RunPackageTests(t, []StringTest{
		{
			"type foo struct{}",
			"local foo\n\nfoo = {}",
		},
		{
			"type foo struct{}; func (f *foo) Bar() int { return 5 }",
			"local foo\n\nfoo = {}\n\nfoo.Bar = function(f)\n\treturn 5\nend",
		},
	})
}
