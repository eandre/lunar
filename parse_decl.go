package lunar

import (
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/types"
	"path/filepath"
)

func (p *Parser) parseDeclNames(d ast.Decl) []string {
	switch t := d.(type) {
	case *ast.FuncDecl:
		return []string{t.Name.Name}
	case *ast.GenDecl:
		names := make([]string, 0, len(t.Specs))
		for _, spec := range t.Specs {
			switch st := spec.(type) {
			case *ast.ImportSpec:
				// Imports should not be in local ns from declarations
			case *ast.ValueSpec:
				for _, n := range st.Names {
					names = append(names, n.Name)
				}
			case *ast.TypeSpec:
				names = append(names, st.Name.Name)
			}
		}
		return names
	default:
		p.errorf(d, "Unhandled Decl type %T", d)
		return nil // Cannot happen
	}
}

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
	w.WriteNewline()
}

func (p *Parser) parseTypeSpec(w *Writer, s *ast.TypeSpec) {
	switch t := s.Type.(type) {
	case *ast.StructType:
		w.WriteLinef("%s = {}", s.Name.Name)
		if s.Name.IsExported() {
			w.WriteLinef("%s.%s = %s", p.pkgName, s.Name.Name, s.Name.Name)
		}
	case *ast.InterfaceType:
		// No need to write anything for interfaces since they are only used for static typing
	default:
		p.errorf(s, "Unhandled TypeSpec type %T", t)
	}
}

func (p *Parser) parseImportSpec(w *Writer, s *ast.ImportSpec) {
	obj := p.importObject(s)
	if pn, ok := obj.(*types.PkgName); ok && p.isTransientPkg(pn.Imported()) {
		return
	}

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
		w.WriteStringf("%s = ", d.Name.Name)
	}

	p.parseFuncLit(w, &ast.FuncLit{
		Type: d.Type,
		Body: d.Body,
	}, recv)
	w.WriteNewline()
	if d.Recv == nil && d.Name.IsExported() {
		w.WriteLinef("%s.%s = %s", p.pkgName, d.Name.Name, d.Name.Name)
	}
	w.WriteNewline()
}
