package lunar

import (
	"testing"
)

func TestAssignStmt(t *testing.T) {
	RunSnippetTests(t, []StringTest{
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
	RunSnippetTests(t, []StringTest{
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
	RunSnippetTests(t, []StringTest{
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
	RunFuncTests(t, []StringTest{
		{
			`foo := []string{"a", "b"}; for a := range foo { println(a) }`,
			"local foo = { \"a\", \"b\" }\nfor a in ipairs(foo) do\n\tprintln(a)\nend",
		},
		{
			`foo := []string{"a", "b"}; for a, b := range foo { println(a, b) }`,
			"local foo = { \"a\", \"b\" }\nfor a, b in ipairs(foo) do\n\tprintln(a, b)\nend",
		},
		{
			`foo := []string{"a", "b"}; for _, b := range foo { println(b) }`,
			"local foo = { \"a\", \"b\" }\nfor _, b in ipairs(foo) do\n\tprintln(b)\nend",
		},
		{
			`foo := []string{"a", "b"}; for range foo { }`,
			"local foo = { \"a\", \"b\" }\nfor _ in ipairs(foo) do\nend",
		},
		{
			`foo := map[int]int{5: 3, 3: 2}; for k, v := range foo { println(k, v) }`,
			"local foo = { [5] = 3, [3] = 2 }\nfor k, v in pairs(foo) do\n\tprintln(k, v)\nend",
		},
	})
}
