package lunar

import (
	"go/ast"
	"strings"
)

func (p *Parser) parseCommentGroup(w *Writer, cg *ast.CommentGroup) {
	if cg == nil {
		return
	}

	for _, c := range cg.List {
		// Handle long-style comments using Lua long-style comments
		if c.Text[0:2] == "/*" {
			if strings.Contains(c.Text, "]=]") {
				p.error(cg, "Cannot handle comment containing ']='")
			}
			w.WriteLinef("--[=[%s]=]", c.Text[2:len(c.Text)-2])
		} else {
			w.WriteLinef("--%s", c.Text[2:])
		}
	}
}

func (p *Parser) parseFile(w *Writer, f *ast.File) {
	for _, decl := range f.Decls {
		p.parseNode(w, decl)
	}
}

func (p *Parser) parsePackage(w *Writer, pkg *ast.Package) {
	for _, f := range pkg.Files {
		p.parseFile(w, f)
	}
}
