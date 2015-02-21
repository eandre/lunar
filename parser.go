package lunar

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/types"
	"io"
	"log"
)

import _ "golang.org/x/tools/go/gcimporter"

type Parser struct {
	info *types.Info
}

func NewParser(path string, fset *token.FileSet, pkg *ast.Package, softFail bool) (*Parser, error) {
	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files {
		files = append(files, f)
	}

	info := new(types.Info)

	var parseErr error
	conf := &types.Config{
		Error: func(err error) {
			e := err.(types.Error)
			if !e.Soft || softFail {
				parseErr = err
			}
		},
	}

	typPkg := types.NewPackage(path, pkg.Name)
	types.NewChecker(conf, fset, typPkg, info).Files(files)
	if parseErr != nil {
		return nil, parseErr
	}
	return &Parser{info}, nil
}

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
	case *ast.FuncDecl:
		p.parseFuncDecl(w, t)
	case *ast.File:
		p.parseFile(w, t)
	case *ast.Package:
		p.parsePackage(w, t)
	default:
		p.log("Unhandled node type %T", t)
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
