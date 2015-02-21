package lunar_test

import (
	"testing"
)

func TestStructType(t *testing.T) {
	RunStringTests(t, []StringTest{
		{
			"type foo struct{}",
			"local foo = {}",
		},
	})
	RunPackageStringTests(t, []StringTest{
		{
			"package foo; type foo struct{}; func (f *foo) Bar() { a := 5 }",
			"local foo = {}\nfoo.Bar = function(f)\n\tlocal a = 5\nend",
		},
	})
}
