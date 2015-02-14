package lunar

import (
	"go/ast"
)

func (p *Parser) parseGenDecl(w *Writer, gd *ast.GenDecl) {
	p.parseCommentGroup(w, gd.Doc)
}
