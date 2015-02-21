package lunar

import (
	"go/ast"
	"go/token"
	"path/filepath"
)

func (p *Parser) parseGenDecl(w *Writer, d *ast.GenDecl) {
	p.parseCommentGroup(w, d.Doc)
	for _, spec := range d.Specs {
		switch d.Tok {
		case token.TYPE:
			p.parseTypeSpec(w, spec.(*ast.TypeSpec))
		case token.IMPORT:
			p.parseImportSpec(w, spec.(*ast.ImportSpec))
		default:
			p.errorf(d, "Unhandled GenDecl token type %q", d.Tok.String())
		}
	}
}

func (p *Parser) parseTypeSpec(w *Writer, s *ast.TypeSpec) {
	switch t := s.Type.(type) {
	case *ast.StructType:
		w.WriteLinef("local %s = {}", s.Name.Name)
	default:
		p.errorf(s, "Unhandled TypeSpec type %T", t)
	}
}

func (d *Parser) parseImportSpec(w *Writer, s *ast.ImportSpec) {
	importPath := s.Path.Value
	importPath = importPath[1 : len(importPath)-1] // Skip surrounding quotes
	pkgName := filepath.Base(importPath)
	localName := pkgName
	if s.Name != nil {
		localName = s.Name.Name
	}
	w.WriteLinef(`local %s = _G["%s"]`, localName, pkgName)
}

func (p *Parser) parseFuncDecl(w *Writer, d *ast.FuncDecl) {
	recv := ""
	if d.Recv != nil {
		recv = d.Recv.List[0].Names[0].Name
		var typeName string
		switch typ := d.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			typeName = typ.X.(*ast.Ident).Name
		default:
			p.errorf(d, "Unhandled FuncDecl with Recv type %T", typ)
		}
		w.WriteStringf("%s.%s = ", typeName, d.Name.Name)
	} else {
		w.WriteStringf("local %s = ", d.Name.Name)
	}

	p.parseFuncLit(w, &ast.FuncLit{
		Type: d.Type,
		Body: d.Body,
	}, recv)
	w.WriteNewline()
}
