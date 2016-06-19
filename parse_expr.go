package lunar

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
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
		p.parseBasicLit(w, t)
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
		p.parseIndexExpr(w, t, false)
	default:
		p.errorf(s, "Unsupported expression type %T", s)
	}
}

func (p *Parser) parseBasicLit(w *Writer, e *ast.BasicLit) {
	switch e.Kind {
	case token.INT, token.FLOAT:
		w.WriteString(e.Value)
	case token.CHAR:
		w.WriteStringf(`"%s"`, e.Value[1:len(e.Value)-1])
	case token.STRING:
		inner := e.Value[1 : len(e.Value)-1]
		if e.Value[0] == '`' {
			w.WriteStringf(`[=[%s]=]`, inner)
		} else {
			w.WriteStringf(`"%s"`, inner)
		}
	default:
		p.errorf(e, "Unsupported basic literal type %s", e.Kind)
	}
}

func (p *Parser) parseBinaryExpr(w *Writer, e *ast.BinaryExpr) {
	isStr := func(x ast.Expr) bool {
		typ := p.exprType(e.X).Underlying()
		b, ok := typ.(*types.Basic)
		return ok && (b.Info()&types.IsString) != 0
	}

	p.parseExpr(w, e.X)
	w.WriteByte(' ')
	switch e.Op {
	// Expressions that are cross-compatible
	case token.SUB, token.MUL, token.QUO, token.REM, token.EQL, token.LSS, token.GTR, token.LEQ, token.GEQ:
		w.WriteString(e.Op.String())
	case token.ADD:
		if isStr(e.X) || isStr(e.Y) {
			w.WriteString("..")
		} else {
			w.WriteString("+")
		}
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

		// Handle add separately
		if e.Op == token.ADD_ASSIGN {
			if isStr(e.X) || isStr(e.Y) {
				w.WriteString("..")
			} else {
				w.WriteString("+")
			}
		} else {
			w.WriteByte(e.Op.String()[0])
		}
	default:
		p.errorf(e, "Got unhandled binary expression token type %q", e.Op.String())
	}
	w.WriteByte(' ')
	p.parseExpr(w, e.Y)
}

func (p *Parser) parseCallExpr(w *Writer, e *ast.CallExpr) {
	if ok := p.parseRaw(w, e); ok {
		return
	}

	// If we have a builtin, handle it separately
	tav := p.exprTypeAndValue(e.Fun)
	if tav.IsBuiltin() {
		p.parseBuiltin(w, e, tav)
		return
	}
	// If we have a type conversion, just unwrap it
	if tav.IsType() {
		w.WriteByte('(')
		p.parseExpr(w, e.Args[0])
		w.WriteByte(')')
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
		lastArg := (i + 1) == narg
		if e.Ellipsis.IsValid() && lastArg {
			w.WriteString("unpack(")
			p.parseExpr(w, arg)
			w.WriteByte(')')
		} else {
			p.parseExpr(w, arg)
		}
		if !lastArg {
			w.WriteString(", ")
		}
	}
	w.WriteByte(')')
}

func (p *Parser) parseCompositeLit(w *Writer, l *ast.CompositeLit) {
	typ := p.exprType(l)
	switch typ := typ.Underlying().(type) {
	case *types.Array, *types.Slice:
		w.WriteString("{ ")
		nel := len(l.Elts)
		for i, el := range l.Elts {
			p.parseExpr(w, el)
			if (i + 1) != nel {
				w.WriteString(", ")
			}
		}
		w.WriteString(" }")

	case *types.Map:
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

	case *types.Struct:
		// Constructor of a type
		// Check if we have methods
		typTyp := p.exprTypeRaw(l.Type)
		mset := types.NewMethodSet(types.NewPointer(typTyp))
		haveMethods := mset.Len() > 0

		if haveMethods {
			w.WriteString("setmetatable({ ")
		} else {
			w.WriteString("{ ")
		}

		initialized := map[string]bool{}
		nel := len(l.Elts)
		for i, el := range l.Elts {
			var value ast.Expr
			var fieldName string
			w.WriteString(`["`)
			if kv, ok := el.(*ast.KeyValueExpr); ok {
				fieldName = kv.Key.(*ast.Ident).Name
				value = kv.Value
			} else {
				fieldName = typ.Field(i).Name()
				value = el
			}
			w.WriteString(getFieldName(typ, fieldName))
			w.WriteString(`"] = `)
			p.parseExpr(w, value)
			if (i + 1) != nel {
				w.WriteString(", ")
			}

			initialized[fieldName] = true
		}

		// Go through all fields that were not initialized and assign default values
		for i := 0; i < typ.NumFields(); i++ {
			field := typ.Field(i)
			if !initialized[field.Name()] {
				if val := p.getZeroValue(w, field.Type(), typ.Tag(i)); val != "nil" {
					if i > 0 || len(initialized) > 0 {
						// We have a previous field; add a preceding comma
						w.WriteString(", ")
					}

					w.WriteStringf(`["%s"] = %s`, computeFieldName(field.Name(), typ.Tag(i)), val)
				}
			}
		}

		if haveMethods {
			w.WriteString(" }, {__index=")
			p.parseExpr(w, l.Type)
			w.WriteString("})")
		} else {
			w.WriteString("}")
		}
	default:
		p.errorf(l, "Unhandled CompositeLit type: %T", typ)
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
		if (i + 1) < nn {
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

func getFieldName(strct *types.Struct, defaultName string) string {
	for i := 0; i < strct.NumFields(); i++ {
		if strct.Field(i).Name() == defaultName {
			return computeFieldName(defaultName, strct.Tag(i))
		}
	}
	return defaultName
}

func computeFieldName(defaultName, tag string) string {
	st := reflect.StructTag(tag)
	if name := st.Get("luaname"); name != "" {
		return name
	}
	return defaultName
}

func (p *Parser) parseSelectorExpr(w *Writer, e *ast.SelectorExpr, inCall bool) {
	if ident, ok := e.X.(*ast.Ident); ok {
		obj := p.identObject(ident)
		if pn, ok := obj.(*types.PkgName); ok {
			if p.IsTransientPkg(pn.Imported()) {
				w.WriteString(e.Sel.Name)
				return
			}
		}
	}

	// See if e.X is a method
	isMethod := false
	selTyp := p.exprType(e.X)
	for {
		if ptr, ok := selTyp.(*types.Pointer); ok {
			selTyp = ptr.Elem()
		} else {
			break
		}
	}
	type hasMethods interface {
		NumMethods() int
		Method(i int) *types.Func
	}
	if obj, ok := selTyp.(hasMethods); ok {
		for i := 0; i < obj.NumMethods(); i++ {
			m := obj.Method(i)
			if m.Name() == e.Sel.Name {
				isMethod = true
				break
			}
		}
	}

	selName := e.Sel.Name
	if !isMethod {
		// Not a method, but is it a field with a tag?
		if strct, ok := selTyp.Underlying().(*types.Struct); ok {
			selName = getFieldName(strct, selName)
		}
	}

	if inCall {
		p.parseExpr(w, e.X)
		if isMethod {
			w.WriteStringf(`:%s`, selName)
		} else {
			w.WriteStringf(`.%s`, selName)
		}
		return
	}

	// If the type is a function and we are referring to a method,
	// this is a method expression.
	if _, ok := p.exprType(e).(*types.Signature); ok && isMethod {
		// Method expression; create a stable closure to preserve equality.
		w.WriteString("builtins.create_closure(")
		p.parseExpr(w, e.X)
		w.WriteStringf(`, "%s")`, selName)
		return
	}

	// Regular field lookup
	p.parseExpr(w, e.X)
	w.WriteStringf(`.%s`, selName)
}

func (p *Parser) parseUnaryExpr(w *Writer, e *ast.UnaryExpr) {
	switch e.Op {
	case token.AND:
		// Taking the address of something is a no-op in lua since we don't have value types
		p.parseExpr(w, e.X)
	case token.NOT:
		w.WriteString("(not ")
		p.parseExpr(w, e.X)
		w.WriteByte(')')
	case token.SUB, token.ADD:
		w.WriteString("(" + e.Op.String())
		p.parseExpr(w, e.X)
		w.WriteByte(')')
	default:
		p.errorf(e, "Unhandled UnaryExpr operand: %v", e.Op)
	}
}

func (p *Parser) parseIndexExpr(w *Writer, e *ast.IndexExpr, assign bool) {
	if !assign {
		w.WriteByte('(')
	}
	typ := p.exprType(e.X).Underlying()
	switch typ.(type) {
	case *types.Map:
		p.parseExpr(w, e.X)
		w.WriteByte('[')
		p.parseExpr(w, e.Index)
		w.WriteByte(']')
	case *types.Slice, *types.Array:
		p.parseExpr(w, e.X)
		w.WriteByte('[')
		p.parseExpr(w, e.Index)
		w.WriteString(" + 1]")
	default:
		p.errorf(e, "unhandled index type %s", typ)
	}

	if !assign {
		w.WriteString(" or ")
		p.writeZeroValue(w, p.exprType(e).Underlying(), "")
		w.WriteByte(')')
	}
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

func (p *Parser) writeZeroValue(w *Writer, typ types.Type, tag string) {
	w.WriteString(p.getZeroValue(w, typ, tag))
}

func (p *Parser) getZeroValue(w *Writer, typ types.Type, tag string) string {
	st := reflect.StructTag(tag)
	if val := st.Get("luadefault"); val != "" {
		return val
	}

	switch typ := typ.(type) {
	case *types.Map:
		return "nil"
	case *types.Basic:
		switch i := typ.Info(); true {
		case (i & types.IsBoolean) != 0:
			return "false"
		case (i & types.IsNumeric) != 0:
			return "0"
		case (i & types.IsString) != 0:
			return `""`
		default:
			panic("Unhandled zero value type")
		}
	default:
		return "nil"
	}
}
