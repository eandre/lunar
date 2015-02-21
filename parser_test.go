package lunar_test

import (
	"bytes"
	"fmt"
	"github.com/eandre/lunar"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func ParseString(snippet string) (string, string, error) {
	src := fmt.Sprintf(`
package dummy
func testFunc() {%s}`, snippet)

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("Could not parse snippet %q: %v", snippet, err))
	}
	pkg, err := ast.NewPackage(fset, map[string]*ast.File{"dummy.go": f}, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("Could not create package: %v", err))
	}

	node := f.Decls[0].(*ast.FuncDecl).Body
	tree := &bytes.Buffer{}
	ast.Fprint(tree, fset, node, nil)

	p, err := lunar.NewParser("dummy", fset, pkg, false)
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

func ParsePackageString(src string) (string, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dummy.go", src, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("Could not parse source %q: %v", src, err))
	}
	pkg, err := ast.NewPackage(fset, map[string]*ast.File{"dummy.go": f}, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("Could not create package: %v", err))
	}

	tree := &bytes.Buffer{}
	ast.Fprint(tree, fset, f, nil)

	p, err := lunar.NewParser("dummy", fset, pkg, false)
	if err != nil {
		return "", tree.String(), err
	}
	buf := &bytes.Buffer{}
	err = p.ParseNode(buf, f)
	if err != nil {
		return "", tree.String(), err
	}
	return strings.TrimSpace(buf.String()), tree.String(), nil
}

func TestParseString(t *testing.T) {
	s, _, err := ParseString("")
	if err != nil {
		t.Fatalf("Could not parse emptry string: %v", err)
	} else if s != "" {
		t.Fatalf("Got lua snippet %q, want %q", s, "")
	}
}

type StringTest struct {
	Go  string
	Lua string
}

func RunStringTests(t *testing.T, tests []StringTest) {
	for i, test := range tests {
		lua, tree, err := ParseString(test.Go)
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

func RunPackageStringTests(t *testing.T, tests []StringTest) {
	for i, test := range tests {
		lua, tree, err := ParsePackageString(test.Go)
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
