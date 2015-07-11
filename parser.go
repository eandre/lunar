package lunar

import _ "golang.org/x/tools/go/gcimporter"

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"log"

	"golang.org/x/tools/go/types"
)

type Parser struct {
	info      *types.Info
	fset      *token.FileSet
	pkgName   string
	transient map[string]bool
}

func NewParser(pkgName string, fset *token.FileSet, files []*ast.File) (*Parser, error) {
	info := &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Implicits: make(map[ast.Node]types.Object),
	}
	conf := new(types.Config)
	_, err := conf.Check(pkgName, fset, files, info)
	if err != nil {
		return nil, err
	}
	return &Parser{
		info:      info,
		fset:      fset,
		pkgName:   pkgName,
		transient: make(map[string]bool),
	}, nil
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

func (p *Parser) exprType(x ast.Expr) types.Type {
	if p.info == nil {
		p.error(x, "No type information received; cannot deduct type")
	}
	return p.info.TypeOf(x).Underlying()
}

func (p *Parser) identObject(i *ast.Ident) types.Object {
	if p.info == nil {
		p.error(i, "No type information received; cannot deduct type")
	}
	return p.info.ObjectOf(i)
}

func (p *Parser) importObject(i *ast.ImportSpec) types.Object {
	if p.info == nil {
		p.error(i, "No type information received; cannot deduct type")
	}
	return p.info.Implicits[i]
}

func (p *Parser) exprTypeAndValue(x ast.Expr) types.TypeAndValue {
	if p.info == nil {
		p.error(x, "No type information received; cannot deduct type")
	}
	return p.info.Types[x]
}

func (p *Parser) MarkTransientPackage(path string) {
	p.transient[path] = true
}

func (p *Parser) isTransientPkg(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return p.transient[pkg.Path()]
}

type ParseError struct {
	node ast.Node
	err  string
}

func (e ParseError) Error() string {
	return e.err
}
