package lunar

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/types"
)

func (p *Parser) parseExpr(w *Writer, s ast.Expr) {
	if s == nil {
		w.WriteString("nil")
		return
	}

	switch t := s.(type) {
	// Simple expression types, handled inline
	case *ast.Ident:
		pkg := p.nodePkg(t)
		obj := pkg.Info.ObjectOf(t)

		// If this itself is a package name, write it outright
		if _, ok := obj.(*types.PkgName); ok {
			w.WriteString(t.Name)
			return
		}

		// Otherwise prepend the package name
		if obj := pkg.Info.Uses[t]; obj != nil && obj.Pkg() != nil {
			// if it's a field, don't prepend package name
			addPkg := true
			switch obj := obj.(type) {
			case *types.Var:
				addPkg = !obj.IsField()
			}
			if addPkg {
				w.WriteString(obj.Pkg().Name() + ".")
			}
		}
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
	case *ast.CompositeLit:
		p.parseCompositeLit(w, t)
	case *ast.FuncLit:
		p.parseFuncLit(w, t, "")
	case *ast.SelectorExpr:
		p.parseSelectorExpr(w, t, false)
	case *ast.UnaryExpr:
		p.parseUnaryExpr(w, t)
	case *ast.IndexExpr:
		p.parseIndexExpr(w, t)
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

	// If we have a builtin, handle it separately
	tav := p.exprTypeAndValue(e.Fun)
	if tav.IsBuiltin() {
		p.parseBuiltin(w, e, tav)
		return
	}

	if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
		p.parseSelectorExpr(w, sel, true)
	} else {
		p.parseExpr(w, e.Fun)
	}

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

func (p *Parser) parseCompositeLit(w *Writer, l *ast.CompositeLit) {
	switch t := l.Type.(type) {
	case *ast.ArrayType:
		w.WriteString("{ ")
		nel := len(l.Elts)
		for i, el := range l.Elts {
			p.parseExpr(w, el)
			if (i + 1) != nel {
				w.WriteString(", ")
			}
		}
		w.WriteString(" }")

	case *ast.MapType:
		w.WriteString("{ ")
		nel := len(l.Elts)
		for i, el := range l.Elts {
			kv := el.(*ast.KeyValueExpr)
			w.WriteByte('[')
			p.parseExpr(w, kv.Key)
			w.WriteString("] = ")
			p.parseExpr(w, kv.Value)
			if (i + 1) != nel {
				w.WriteString(", ")
			}
		}
		w.WriteString(" }")

	case *ast.Ident:
		// Constructor of a type
		w.WriteString("setmetatable({ ")
		nel := len(l.Elts)
		for i, el := range l.Elts {
			kv := el.(*ast.KeyValueExpr)
			w.WriteString(`["`)
			p.parseExpr(w, kv.Key)
			w.WriteString(`"] = `)
			p.parseExpr(w, kv.Value)
			if (i + 1) != nel {
				w.WriteString(", ")
			}
		}
		w.WriteString(" }, {__index=")
		p.parseExpr(w, t)
		w.WriteString("})")
	default:
		p.errorf(l, "Unhandled CompositeLit type: %T", t)
	}
}

func (p *Parser) parseFuncLit(w *Writer, f *ast.FuncLit, recv string) {
	w.WriteString("function(")
	params := f.Type.Params.List
	np := len(params)

	if recv != "" {
		w.WriteString(recv)
		if np > 0 {
			w.WriteString(", ")
		}
	}

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

func (p *Parser) parseSelectorExpr(w *Writer, e *ast.SelectorExpr, method bool) {
	if ident, ok := e.X.(*ast.Ident); ok {
		obj := p.identObject(ident)
		if pn, ok := obj.(*types.PkgName); ok && p.IsTransientPkg(pn.Imported()) {
			w.WriteString(e.Sel.Name)
			return
		}
	}
	p.parseExpr(w, e.X)
	if method {
		w.WriteStringf(`:%s`, e.Sel.Name)
	} else {
		w.WriteStringf(`.%s`, e.Sel.Name)
	}
}

func (p *Parser) parseUnaryExpr(w *Writer, e *ast.UnaryExpr) {
	switch e.Op {
	case token.AND:
		// Taking the address of something is a no-op in lua since we don't have value types
		p.parseExpr(w, e.X)
	default:
		p.errorf(e, "Unhandled UnaryExpr operand: %v", e.Op)
	}
}

func (p *Parser) parseIndexExpr(w *Writer, e *ast.IndexExpr) {
	p.parseExpr(w, e.X)
	w.WriteByte('[')
	p.parseExpr(w, e.Index)
	w.WriteString(" + 1]")
}
