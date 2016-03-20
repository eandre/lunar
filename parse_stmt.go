package lunar

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (p *Parser) parseBlockStmt(w *Writer, b *ast.BlockStmt) {
	for _, stmt := range b.List {
		p.parseStmt(w, stmt)
	}
}

func (p *Parser) parseStmt(w *Writer, s ast.Stmt) {
	switch t := s.(type) {
	case *ast.AssignStmt:
		p.parseAssignStmt(w, t)
	case *ast.BlockStmt:
		p.parseBlockStmt(w, t)
	case *ast.DeclStmt:
		p.parseDeclStmt(w, t)
	case *ast.ExprStmt:
		// ExprStmt is an expression that ends in a newline
		p.parseExpr(w, t.X)
		w.WriteBytes(newline)
	case *ast.ReturnStmt:
		p.parseReturnStmt(w, t)
	case *ast.IfStmt:
		p.parseIfStmt(w, t)
	case *ast.RangeStmt:
		p.parseRangeStmt(w, t)
	default:
		p.errorf(s, "Unhandled statement type %T", t)
	}
}

func (p *Parser) parseAssignStmt(w *Writer, s *ast.AssignStmt) {
	nl := len(s.Lhs)
	nr := len(s.Rhs)

	for _, lhs := range s.Lhs {
		_, ident := lhs.(*ast.Ident)
		_, index := lhs.(*ast.IndexExpr)
		_, sel := lhs.(*ast.SelectorExpr)
		if !ident && !index && !sel {
			p.errorf(s, "Got assignment to non-identifier/index/selector %T", lhs)
		}
	}

	switch s.Tok {
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
		// combined assignment and binary expression; handle separately
		if nl != 1 || nr != 1 {
			p.errorf(s, "Got assignment with token %q and != 1 expr per side (%d vs %d)", s.Tok.String(), nl, nr)
		}

		// Left hand side appears twice
		p.parseExpr(w, s.Lhs[0])
		w.WriteString(" = ")
		p.parseExpr(w, s.Lhs[0])

		// Add first part of operator
		w.WriteByte(' ')
		w.WriteByte(s.Tok.String()[0])
		w.WriteByte(' ')

		// Add right hand side
		p.parseExpr(w, s.Rhs[0])
		w.WriteBytes(newline)
		return

	case token.DEFINE:
		// combined assignment and declaration, prepend "local"
		w.WriteString("local ")
	}

	for i, lhs := range s.Lhs {
		switch lhs := lhs.(type) {
		case *ast.IndexExpr:
			p.parseIndexExpr(w, lhs, true)
		default:
			p.parseExpr(w, lhs)
		}

		// Add a comma if we have more expressions coming
		if (i + 1) != nl {
			w.WriteString(", ")
		}
	}
	w.WriteString(" = ")
	for i, rhs := range s.Rhs {
		// TODO(eandre) Need to map this to the zero value for each type instead of "nil"
		p.parseExpr(w, rhs)
		// Add a comma if we have more expressions coming
		if (i + 1) != nr {
			w.WriteString(", ")
		}
	}
	w.WriteBytes(newline)
}

func (p *Parser) parseDeclStmt(w *Writer, s *ast.DeclStmt) {
	p.parseGenDecl(w, s.Decl.(*ast.GenDecl), false)
}

func (p *Parser) parseReturnStmt(w *Writer, r *ast.ReturnStmt) {
	// Naked return
	if r.Results == nil {
		w.WriteLine("return")
		return
	}

	w.WriteString("return ")
	nr := len(r.Results)
	for i, res := range r.Results {
		p.parseExpr(w, res)
		if (i + 1) != nr {
			w.WriteString(", ")
		}
	}
	w.WriteNewline()
}

func (p *Parser) parseIfStmt(w *Writer, s *ast.IfStmt) {
	w.WriteString("if ")
	p.parseExpr(w, s.Cond)
	w.WriteString(" then")
	w.WriteNewline()
	w.Indent()
	p.parseBlockStmt(w, s.Body)
	w.Dedent()
	if s.Else != nil {
		if elif, ok := s.Else.(*ast.IfStmt); ok {
			// We have an else-if statement
			w.WriteString("else")
			p.parseIfStmt(w, elif)
			return
		}

		// Regular else statement
		w.WriteLine("else")
		w.Indent()
		p.parseStmt(w, s.Else)
		w.Dedent()
	}
	w.WriteLine("end")
}

func (p *Parser) parseRangeStmt(w *Writer, s *ast.RangeStmt) {
	// TODO(eandre) We can only handle ":=" range statements for now, since
	// Lua uses a local scope in for loops. To get around this to allow for
	// outer-declared variables we'd have to allocate a temporary variable
	// and resassign the outer variable at the start of the loop.
	// ILLEGAL is supported also, which is the case where no variables at all
	// are assigned to.
	if s.Tok != token.DEFINE && s.Tok != token.ILLEGAL {
		p.errorf(s, "Unhandled range token %s", s.Tok.String())
	}

	w.WriteString("for ")

	// Lua requires at least one local variable; if we don't have one
	// use "_"
	if s.Key == nil {
		w.WriteString("_")
	} else {
		w.WriteString(s.Key.(*ast.Ident).Name)
	}

	if s.Value != nil {
		w.WriteStringf(", %s", s.Value.(*ast.Ident).Name)
	}

	w.WriteString(" in ")

	switch t := p.exprType(s.X).(type) {
	case *types.Slice:
		w.WriteString("ipairs(")
		p.parseExpr(w, s.X)
		// Add "or {}" to match Go's behavior of iteration over nil slices
		w.WriteString(" or {})")
	case *types.Map:
		w.WriteString("pairs(")
		p.parseExpr(w, s.X)
		// Add "or {}" to match Go's behavior of iteration over nil maps
		w.WriteString(" or {})")
	default:
		p.errorf(s, "Unhandled RangeStmt expression type %T", t)
	}

	w.WriteString(" do")
	w.WriteNewline()
	w.Indent()
	p.parseBlockStmt(w, s.Body)
	w.Dedent()
	w.WriteLine("end")
}
