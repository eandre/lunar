package lunar

import (
	"testing"
)

func TestStructType(t *testing.T) {
	RunPackageTests(t, []StringTest{
		{
			"type foo struct{}",
			"local foo = {}",
		},
	})
	RunPackageTests(t, []StringTest{
		{
			"type foo struct{}; func (f *foo) Bar() int { return 5 }",
			"local foo = {}\nfoo.Bar = function(f)\n\treturn 5\nend",
		},
	})
}
