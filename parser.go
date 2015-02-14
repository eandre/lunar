package lunar

import (
	"fmt"
	"go/ast"
	"io"
	"log"
)

type Parser struct{}

func (p *Parser) ParseNode(w io.Writer, n ast.Node) (err error) {
	// Handle panics
	defer func() {
		e := recover()
		if e != nil {
			switch e.(type) {
			case ParseError, WriteError:
				err = e.(error)
				return
			}

			// Panic again if it wasn't a parse error
			panic(e)
		}
	}()

	writer := NewWriter(w)
	p.parseNode(writer, n)
	return nil
}

func (p *Parser) parseNode(w *Writer, n ast.Node) {
	switch t := n.(type) {
	case *ast.GenDecl:
		p.parseGenDecl(w, t)
	case *ast.BlockStmt:
		p.parseBlockStmt(w, t)
	default:
		p.log("Unhandled node type %v", n)
	}
}

func (p *Parser) log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (p *Parser) error(node ast.Node, err string) {
	panic(ParseError{node: node, err: err})
}

func (p *Parser) errorf(node ast.Node, format string, args ...interface{}) {
	err := fmt.Sprintf(format, args...)
	p.error(node, err)
}

type ParseError struct {
	node ast.Node
	err  string
}

func (e ParseError) Error() string {
	return e.err
}
