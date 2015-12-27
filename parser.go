package lunar

import (
	"fmt"
	"go/ast"
	"io"
	"log"

	"golang.org/x/tools/go/types"
	"golang.org/x/tools/go/loader"
)

type Parser struct {
	prog *loader.Program
	transient map[string]bool
	testPkgName string // for testing purposes
}

func NewParser(prog *loader.Program) *Parser {
	return &Parser{
		prog: prog,
		transient: make(map[string]bool),
	}
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
	p.parseNode(writer, n, true)
	return nil
}

func (p *Parser) parseNode(w *Writer, n ast.Node, topLevel bool) {
	switch t := n.(type) {
	case *ast.GenDecl:
		p.parseGenDecl(w, t, topLevel)
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
	pkg := p.nodePkg(x)
	if typ := pkg.Info.TypeOf(x); typ != nil {
		return typ.Underlying()
	}
	p.error(x, "Could not determine type of expr")
	return nil // unreachable
}

func (p *Parser) identObject(i *ast.Ident) types.Object {
	pkg := p.nodePkg(i)
	if obj := pkg.Info.ObjectOf(i); obj != nil {
		return obj
	}
	p.error(i, "Could not determine ident object")
	return nil // unreachable
}

func (p *Parser) importObject(i *ast.ImportSpec) types.Object {
	pkg := p.nodePkg(i)
	if obj := pkg.Info.Implicits[i]; obj != nil {
		return obj
	}
	p.error(i, "Could not determine import object")
	return nil // unreachable
}

func (p *Parser) exprTypeAndValue(x ast.Expr) types.TypeAndValue {
	if p.testPkgName != "" {
		// for testing purposes
		return types.TypeAndValue{}
	}

	pkg := p.nodePkg(x)
	if tav, ok := pkg.Info.Types[x]; ok {
		return tav
	}
	p.error(x, "Could not determine type and value of expr")
	return types.TypeAndValue{} // unreachable
}

func (p *Parser) pkgName(n ast.Node) string {
	if p.testPkgName != "" {
		// for testing purposes
		return p.testPkgName
	}

	pkg := p.nodePkg(n)
	return pkg.Pkg.Name()
}

func (p *Parser) nodePkg(n ast.Node) *loader.PackageInfo {
	pkg, _, _ := p.prog.PathEnclosingInterval(n.Pos(), n.End())
	if pkg == nil {
		p.error(n, "Could not get package for node")
	}
	return pkg
}

func (p *Parser) MarkTransientPackage(path string) {
	p.transient[path] = true
}

func (p *Parser) IsTransientPkg(pkg *types.Package) bool {
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
