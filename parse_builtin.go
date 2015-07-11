package lunar

import (
	"go/ast"

	"golang.org/x/tools/go/types"
)

func (p *Parser) parseBuiltin(w *Writer, e *ast.CallExpr, tav types.TypeAndValue) {
	id := e.Fun.(*ast.Ident)
	switch id.Name {
	case "make":
		w.WriteString("builtins.make(<type>, ")
		p.parseExpr(w, e.Args[1])
		w.WriteByte(')')
	default:
		p.errorf(e, "Unhandled builtin %s", id.Name)
	}
}
