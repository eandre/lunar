package lunar

import (
	"go/ast"
	"go/types"
)

const LuaPkgPath = "github.com/eandre/lunar/lua"

func (p *Parser) parseRaw(w *Writer, e *ast.CallExpr) (ok bool) {
	sel, ok := e.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Raw" {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	pkg, ok := p.identObject(ident).(*types.PkgName)
	if !ok {
		return false
	}

	if pkg.Imported().Path() != LuaPkgPath {
		return
	}

	for _, arg := range e.Args {
		switch arg := arg.(type) {
		case *ast.BasicLit:
			if arg.Value[0] == '"' || arg.Value[0] == '`' {
				// String literal; skip quotes
				w.WriteString(arg.Value[1 : len(arg.Value)-1])
			} else {
				w.WriteString(arg.Value)
			}
		default:
			p.parseExpr(w, arg)
		}
	}
	return true
}
