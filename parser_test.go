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
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("Could not parse snippet %q: %v", snippet, err))
	}

	node := f.Decls[0].(*ast.FuncDecl).Body
	tree := &bytes.Buffer{}
	ast.Fprint(tree, fset, node, nil)

	buf := &bytes.Buffer{}
	p := &lunar.Parser{}
	err = p.ParseNode(buf, node)
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
			t.Errorf("%d. Go %q resulted in error: %v", i, test.Go, err)
			continue
		} else if lua != test.Lua {
			t.Logf("Got tree: %s", tree)
			t.Errorf("%d. Go %q resulted in Lua %q; want %q", i, test.Go, lua, test.Lua)
			continue
		}
	}
}
