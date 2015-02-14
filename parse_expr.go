package lunar

import (
	"go/ast"
	"go/token"
)

func (p *Parser) parseExpr(w *Writer, s ast.Expr) {
	switch t := s.(type) {
	// Simple expression types, handled inline
	case *ast.Ident:
		w.WriteString(t.Name)
	case *ast.BasicLit:
		w.WriteString(t.Value) // TODO(eandre) Assume basic literals are cross-compatible?
	case *ast.ParenExpr:
		w.WriteByte('(')
		p.parseExpr(w, t.X)
		w.WriteByte(')')

	// More complex expression types, handled separately
	case *ast.BinaryExpr:
		p.parseBinaryExpr(w, t)
	case *ast.CallExpr:
		p.parseCallExpr(w, t)
	case *ast.FuncLit:
		p.parseFuncLit(w, t)

	default:
		p.errorf(s, "Unsupported expression type %T", s)
	}
}

func (p *Parser) parseBinaryExpr(w *Writer, e *ast.BinaryExpr) {
	p.parseExpr(w, e.X)
	w.WriteByte(' ')
	switch e.Op {
	// Expressions that are cross-compatible
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.EQL, token.LSS, token.GTR, token.LEQ, token.GEQ:
		w.WriteString(e.Op.String())
	case token.NOT:
		w.WriteString("not")
	case token.NEQ:
		w.WriteString("~=")
	case token.LOR:
		w.WriteString("or")
	case token.LAND:
		w.WriteString("and")
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
		// Make sure the left operator is of type Ident
		if _, ok := e.X.(*ast.Ident); !ok {
			p.errorf(e, "LHS of %s is type %T, not Ident", e.Op, e.X)
		}
		w.WriteString(" = ")
		p.parseExpr(w, e.X)
		w.WriteByte(e.Op.String()[0])
	default:
		p.errorf(e, "Got unhandled binary expression token type %q", e.Op.String())
	}
	w.WriteByte(' ')
	p.parseExpr(w, e.Y)
}

func (p *Parser) parseCallExpr(w *Writer, e *ast.CallExpr) {
	// TODO(eandre) Handle ellipsis?
	if e.Ellipsis != token.NoPos {
		p.error(e, "CallExpr includes ellipsis (unsupported)")
	}
	p.parseExpr(w, e.Fun)
	w.WriteByte('(')
	narg := len(e.Args)
	for i, arg := range e.Args {
		p.parseExpr(w, arg)
		if (i + 1) != narg {
			w.WriteString(", ")
		}
	}
	w.WriteByte(')')
}

func (p *Parser) parseFuncLit(w *Writer, f *ast.FuncLit) {
	w.WriteString("function(")
	params := f.Type.Params.List
	np := len(params)
	for i, p := range params {
		w.WriteString(p.Names[0].Name)
		if (i + 1) != np {
			w.WriteString(", ")
		}
	}
	w.WriteByte(')')
	w.WriteNewline()
	w.Indent()
	p.parseBlockStmt(w, f.Body)
	w.Dedent()
	w.WriteString("end")
}
