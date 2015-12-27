package lunar

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/types"
)

func (p *Parser) parseGenDecl(w *Writer, d *ast.GenDecl, topLevel bool) {
	p.parseCommentGroup(w, d.Doc)
	for _, spec := range d.Specs {
		switch d.Tok {
		case token.TYPE:
			p.parseTypeSpec(w, spec.(*ast.TypeSpec))
		case token.IMPORT:
			p.parseImportSpec(w, spec.(*ast.ImportSpec))
		case token.CONST:
			p.parseValueSpec(w, spec.(*ast.ValueSpec), topLevel)
		case token.VAR:
			p.parseValueSpec(w, spec.(*ast.ValueSpec), topLevel)
		default:
			p.errorf(d, "Unhandled GenDecl token type %q", d.Tok.String())
		}
	}
	w.WriteNewline()
}

func (p *Parser) parseTypeSpec(w *Writer, s *ast.TypeSpec) {
	switch t := s.Type.(type) {
	case *ast.StructType:
		w.WriteLinef("_%s.%s = {}", p.pkgName(s), s.Name.Name)
	case *ast.InterfaceType:
		// No need to write anything for interfaces since they are only used for static typing
	default:
		p.errorf(s, "Unhandled TypeSpec type %T", t)
	}
}

func (p *Parser) parseImportSpec(w *Writer, s *ast.ImportSpec) {
	obj := p.importObject(s)
	if pn, ok := obj.(*types.PkgName); ok && p.IsTransientPkg(pn.Imported()) {
		return
	}

	pkg := p.nodePkg(s)
	pkgName := pkg.Implicits[s].(*types.PkgName)
	importPath := s.Path.Value
	importPath = importPath[1 : len(importPath)-1] // Skip surrounding quotes

	var localName string
	// If we have a local name, use that. Dot imports are handled by the fact
	// that all raw idents will get the package name prepended, so dot
	// imports just use the standard name.
	if s.Name != nil && s.Name.Name != "." {
		localName = s.Name.Name
	} else {
		// Otherwise use the package name of the actual package being imported
		localName = pkgName.Imported().Name()
	}
	w.WriteLinef(`local _%s = _G["%s"]`, localName, importPath)
}

func (p *Parser) parseValueSpec(w *Writer, s *ast.ValueSpec, topLevel bool) {
	pkgName := p.pkgName(s)
	for i, name := range s.Names {
		var val ast.Expr
		if len(s.Values) > i {
			val = s.Values[i]
		}

		if topLevel {
			w.WriteStringf("_%s.%s = ", pkgName, name)
		} else {
			w.WriteStringf("local %s = ", name)
		}

		if val != nil {
			p.parseExpr(w, val)
			w.WriteNewline()
		} else {
			p.parseZeroValue(w, s.Type)
		}
	}
}

func (p *Parser) parseFuncDecl(w *Writer, d *ast.FuncDecl) {
	pkgName := p.pkgName(d)
	recv := ""
	isInit := false
	if d.Recv != nil {
		recv = d.Recv.List[0].Names[0].Name
		var typeName string
		switch typ := d.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			typeName = typ.X.(*ast.Ident).Name
		default:
			p.errorf(d, "Unhandled FuncDecl with Recv type %T", typ)
		}
		w.WriteStringf("_%s.%s.%s = ", pkgName, typeName, d.Name.Name)
	} else if d.Name.Name == "init" {
		// init function; handle specially
		isInit = true
		w.WriteString("builtins.add_init(")
	} else {
		w.WriteStringf("_%s.%s = ", pkgName, d.Name.Name)
	}

	p.parseFunc(w, d.Type, d.Body, recv, d.Name)
	if isInit {
		w.WriteByte(')')
	}

	w.WriteNewline()
	w.WriteNewline()
}
