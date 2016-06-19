package lunar

import (
	"go/ast"
	"go/types"
)

func (p *Parser) parseBuiltin(w *Writer, e *ast.CallExpr, tav types.TypeAndValue) {
	id := e.Fun.(*ast.Ident)
	switch id.Name {
	case "make":
		typ := p.exprType(e.Args[0])
		switch typ := typ.Underlying().(type) {
		case *types.Map:
			w.WriteString("{}")
		case *types.Slice:
			w.WriteString("builtins.makeSlice(function() return ")
			p.writeZeroValue(w, typ.Elem(), "")
			w.WriteString(" end")
			if len(e.Args) > 1 {
				w.WriteString(", ")
				p.parseExpr(w, e.Args[1])
			}
			w.WriteByte(')')
		default:
			p.errorf(e, "Unknown make() type %s", typ)
		}

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

	case "append":
		w.WriteString("builtins.append(")
		for i, arg := range e.Args {
			p.parseExpr(w, arg)
			if (i + 1) < len(e.Args) {
				w.WriteString(", ")
			}
		}
		w.WriteByte(')')

	case "delete":
		w.WriteString("builtins.delete(")
		for i, arg := range e.Args {
			p.parseExpr(w, arg)
			if (i + 1) < len(e.Args) {
				w.WriteString(", ")
			}
		}
		w.WriteByte(')')

	case "panic":
		w.WriteString("error(")
		p.parseExpr(w, e.Args[0])
		w.WriteByte(')')

	case "len":
		typ := p.exprType(e.Args[0])
		switch typ.Underlying().(type) {
		case *types.Map:
			w.WriteString("builtins.mapLength(")
			p.parseExpr(w, e.Args[0])
			w.WriteByte(')')
		default:
			w.WriteString("builtins.length(")
			p.parseExpr(w, e.Args[0])
			w.WriteByte(')')
		}
	default:
		p.errorf(e, "Unhandled builtin %s", id.Name)
	}
}
