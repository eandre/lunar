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
			w.WriteString("_" + t.Name)
			return
		}

		// Otherwise prepend the package name
		if obj := pkg.Info.Uses[t]; obj != nil && obj.Pkg() != nil {
			// if it's a field, don't prepend package name
			addPkg := true
			if p.isFuncLocal(obj) {
				addPkg = false
			}
			switch obj := obj.(type) {
			case *types.Var:
				if obj.IsField() {
					addPkg = false
				}
			}

			if addPkg {
				w.WriteString("_" + obj.Pkg().Name() + ".")
			}
		}
		w.WriteString(t.Name)
	case *ast.BasicLit:
		w.WriteString(t.Value) // TODO(eandre) Assume basic literals are cross-compatible?
	case *ast.ParenExpr:
		w.WriteByte('(')
		p.parseExpr(w, t.X)
		w.WriteByte(')')
	case *ast.TypeAssertExpr:
		p.parseExpr(w, t.X) // noop for now

	// More complex expression types, handled separately
	case *ast.BinaryExpr:
		p.parseBinaryExpr(w, t)
	case *ast.CallExpr:
		p.parseCallExpr(w, t)
	case *ast.CompositeLit:
		p.parseCompositeLit(w, t)
	case *ast.FuncLit:
		p.parseFunc(w, t.Type, t.Body, "", nil)
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

func (p *Parser) parseFunc(w *Writer, typ *ast.FuncType, body *ast.BlockStmt, recv string, declName *ast.Ident) {
	w.WriteString("function(")
	params := typ.Params.List

	if recv != "" {
		w.WriteString(recv)
		if len(params) > 0 {
			w.WriteString(", ")
		}
	}

	var names []string
	for _, p := range params {
		for _, name := range p.Names {
			names = append(names, name.Name)
		}
	}

	pkg := p.nodePkg(typ)
	var sig *types.Signature
	if declName != nil {
		sig = pkg.Defs[declName].Type().(*types.Signature)
	} else {
		sig = pkg.Types[typ].Type.(*types.Signature)
	}

	nn := len(names)
	for i, name := range names {
		if (i+1) < nn {
			w.WriteString(name + ", ")
		} else if sig.Variadic() {
			w.WriteString("...")
		} else {
			w.WriteString(name)
		}
	}

	w.WriteByte(')')
	w.WriteNewline()
	w.Indent()
	if sig.Variadic() {
		w.WriteLinef("local %s = {...}", names[nn-1])
	}
	p.parseBlockStmt(w, body)
	w.Dedent()
	w.WriteString("end")
}

func (p *Parser) parseSelectorExpr(w *Writer, e *ast.SelectorExpr, method bool) {
	if ident, ok := e.X.(*ast.Ident); ok {
		obj := p.identObject(ident)
		if pn, ok := obj.(*types.PkgName); ok {
			method = false // if it's a package name this is not a method call
			if p.IsTransientPkg(pn.Imported()) {
				w.WriteString(e.Sel.Name)
				return
			}
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

func (p *Parser) isFuncLocal(obj types.Object) bool {
	_, path, _ := p.prog.PathEnclosingInterval(obj.Pos(), obj.Pos())
	for _, n := range path {
		switch n.(type) {
		case *ast.FieldList, *ast.Field, *ast.DeclStmt, *ast.AssignStmt, *ast.RangeStmt:
			return true
		}
	}
	return false
}

func (p *Parser) parseZeroValue(w *Writer, typ ast.Expr) {
	switch typ.(type) {
	case *ast.ArrayType, *ast.MapType:
		w.WriteString("{}")
	default:
		w.WriteString("nil")
	}
}
