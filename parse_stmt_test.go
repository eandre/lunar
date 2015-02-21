package lunar_test

import (
	"testing"
)

func TestAssignStmt(t *testing.T) {
	RunStringTests(t, []StringTest{
		// Single assignment
		{
			"a := 5",
			"local a = 5",
		},
		{
			"a = 5",
			"a = 5",
		},

		// Multiple assignment
		{
			"a, b = 5",
			"a, b = 5",
		},
		{
			"a, b = 5, \"hello, world\"",
			"a, b = 5, \"hello, world\"",
		},
		{
			"a, b := 5, \"hello, world\"",
			"local a, b = 5, \"hello, world\"",
		},

		// Assign variable
		{
			"a, b = b, a",
			"a, b = b, a",
		},

		// Assign with math operators
		{
			"a += b",
			"a = a + b",
		},
		{
			"a -= b",
			"a = a - b",
		},
		{
			"a *= b",
			"a = a * b",
		},
		{
			"a /= b",
			"a = a / b",
		},
		{
			"a %= b",
			"a = a % b",
		},
	})
}

func TestReturnStmt(t *testing.T) {
	RunStringTests(t, []StringTest{
		{
			"return",
			"return",
		},
		{
			"return 5",
			"return 5",
		},
		{
			"return 3, 2",
			"return 3, 2",
		},
	})
}

func TestIfStmt(t *testing.T) {
	RunStringTests(t, []StringTest{
		{
			`if a < 3 {
				foo(a)
			}`,
			"if a < 3 then\n\tfoo(a)\nend",
		},
		{
			`if a < 3 || b > 5 {
				foo(a)
			} else if a == 3 {
				bar(b)
			} else if a == 2 {
				bar(a)
			} else {
				moo()
			}`,
			`if a < 3 or b > 5 then
	foo(a)
elseif a == 3 then
	bar(b)
elseif a == 2 then
	bar(a)
else
	moo()
end`,
		},
	})
}

func TestRangeStmt(t *testing.T) {
	RunStringTests(t, []StringTest{
		{
			"for a = range foo { }",
			"for a in pairs(foo)\nend",
		},
		{
			"for a, b = range foo { }",
			"for a, b in pairs(foo)\nend",
		},
	})
}
