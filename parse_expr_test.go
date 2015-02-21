package lunar

import (
	"testing"
)

func TestExpr(t *testing.T) {
	RunSnippetTests(t, []StringTest{
		// Basic literals
		{
			"5.32",
			"5.32",
		},
		{
			`"hello, world"`,
			`"hello, world"`,
		},

		// Binary expressions
		{
			"5 * 3",
			"5 * 3",
		},
		{
			"5 * (3 + 2)",
			"5 * (3 + 2)",
		},
		{
			"a != b",
			"a ~= b",
		},

		// Function calls
		{
			"foo()",
			"foo()",
		},
		{
			"foo(5)",
			"foo(5)",
		},
		{
			`foo(5, "hello, world!")`,
			`foo(5, "hello, world!")`,
		},
		{
			`foo(bar(3, "hi"), baz())`,
			`foo(bar(3, "hi"), baz())`,
		},

		// Anonymous function call
		{
			"(func(foo int) int { return foo })(5)",
			"(function(foo)\n\treturn foo\nend)(5)",
		},
	})
}
