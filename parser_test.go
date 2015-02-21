package lunar

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func ParseSnippet(snippet string) (string, string, error) {
	src := fmt.Sprintf("package dummy\nfunc testFunc() {%s}", snippet)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("Could not parse source %q: %v", src, err))
	}

	tree := &bytes.Buffer{}
	node := f.Decls[0].(*ast.FuncDecl).Body
	ast.Fprint(tree, fset, node, nil)

	p := &Parser{}
	buf := &bytes.Buffer{}
	err = p.ParseNode(buf, node)
	if err != nil {
		return "", tree.String(), err
	}
	return strings.TrimSpace(buf.String()), tree.String(), nil
}

func ParseFunc(src string) (string, string, error) {
	src = fmt.Sprintf(`func testFunc() {%s}`, src)
	get := func(f *ast.File) ast.Node {
		return f.Decls[0].(*ast.FuncDecl).Body
	}
	return parseStr(src, get)
}

func ParsePackage(src string) (string, string, error) {
	get := func(f *ast.File) ast.Node {
		return f
	}
	return parseStr(src, get)
}

func parseStr(src string, get func(*ast.File) ast.Node) (string, string, error) {
	src = "package dummy\n" + src
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("Could not parse source %q: %v", src, err))
	}

	tree := &bytes.Buffer{}
	node := get(f)
	ast.Fprint(tree, fset, node, nil)

	p, err := NewParser("dummy", fset, []*ast.File{f})
	if err != nil {
		return "", tree.String(), err
	}
	buf := &bytes.Buffer{}
	err = p.ParseNode(buf, node)
	if err != nil {
		return "", tree.String(), err
	}
	return strings.TrimSpace(buf.String()), tree.String(), nil
}

type StringTest struct {
	Go  string
	Lua string
}

func RunSnippetTests(t *testing.T, tests []StringTest) {
	for i, test := range tests {
		lua, tree, err := ParseSnippet(test.Go)
		if err != nil {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in error: %#v", i, test.Go, err)
			continue
		} else if lua != test.Lua {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in Lua %q; want %q", i, test.Go, lua, test.Lua)
			continue
		}
	}
}

func RunFuncTests(t *testing.T, tests []StringTest) {
	for i, test := range tests {
		lua, tree, err := ParseFunc(test.Go)
		if err != nil {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in error: %#v", i, test.Go, err)
			continue
		} else if lua != test.Lua {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in Lua %q; want %q", i, test.Go, lua, test.Lua)
			continue
		}
	}
}

func RunPackageTests(t *testing.T, tests []StringTest) {
	for i, test := range tests {
		lua, tree, err := ParsePackage(test.Go)
		if err != nil {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in error: %v", i, test.Go, err)
			continue
		} else if lua != test.Lua {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in Lua %q; want %q", i, test.Go, lua, test.Lua)
			continue
		}
	}
}
