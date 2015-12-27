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
	case "println":
		w.WriteString("print(")
		for i, arg := range e.Args {
			p.parseExpr(w, arg)
			if (i + 1) < len(e.Args) {
				w.WriteString(", ")
			}
		}
		w.WriteByte(')')
	case "print":
		w.WriteString("write(")
		for i, arg := range e.Args {
			p.parseExpr(w, arg)
			if (i + 1) < len(e.Args) {
				w.WriteString(", ")
			}
		}
		w.WriteByte(')')
	default:
		p.errorf(e, "Unhandled builtin %s", id.Name)
	}
}
