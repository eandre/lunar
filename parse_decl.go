package lunar

import (
	"go/ast"
	"go/token"
	"go/types"
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
		p.parseStructType(w, t, s)
	case *ast.InterfaceType, *ast.FuncType, *ast.Ident:
		// No need to write anything since they are only used for static typing
	default:
		p.errorf(s, "Unhandled TypeSpec type %T", t)
	}
}

func (p *Parser) parseImportSpec(w *Writer, s *ast.ImportSpec) {
	pkg := p.nodePkg(s)
	if pkg.Implicits[s] == nil {
		// Anonymous import
		return
	}

	obj := p.importObject(s)
	if pn, ok := obj.(*types.PkgName); ok && p.IsTransientPkg(pn.Imported()) {
		return
	}

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
		} else {
			typ := p.exprType(name)
			p.writeZeroValue(w, typ.Underlying(), "")
		}
		w.WriteNewline()
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

func (p *Parser) parseStructType(w *Writer, t *ast.StructType, s *ast.TypeSpec) {
	pkgName := p.pkgName(s)
	w.WriteLinef("_%s.%s = {}", pkgName, s.Name.Name)

	// Introduce a per-type helper that can initialize structs from a table
	{
		w.WriteLinef(`
_%s.%s._createFromTable = function(tbl)
	if tbl == nil then
		return nil, nil
	end
	if type(tbl) ~= "table" then
		return nil, builtins.create_error("cannot initialize struct from non-table")
	end
	local self = setmetatable({}, {__index=_%s.%s})
	local obj, err
	`, pkgName, s.Name.Name, pkgName, s.Name.Name)

		// For each field, add an initializer
		obj := p.identObject(s.Name)
		typ := obj.Type().Underlying().(*types.Struct)
		for i := 0; i < typ.NumFields(); i++ {
			// Get the raw field type
			f := typ.Field(i)
			fType := f.Type()
			var named *types.Named
			for {
				if n, ok := fType.(*types.Named); named == nil && ok {
					named = n
				}

				if ptr, ok := fType.Underlying().(*types.Pointer); ok {
					fType = ptr.Elem()
				} else {
					break
				}
			}

			name := f.Name()
			switch fType := fType.Underlying().(type) {
			case *types.Struct:
				if named != nil {
					w.WriteLinef(`
		obj, err = _%s.%s._createFromTable(tbl.%s)
		if err ~= nil then
			return nil, err
		end
		self.%s = obj
					`, named.Obj().Pkg().Name(), named.Obj().Name(), name, name)
				} else {
					w.WriteLinef("self.%s = tbl.%s", name, name)
				}
			case *types.Interface:
				// do nothing, can't deserialize interface types since we don't know
				// which concrete type to use.
			default:
				w.WriteLinef("\tself.%s = %s", name, p.getZeroValue(w, fType, ""))
				w.WriteLinef("\tif type(self.%s) == type(tbl.%s) then self.%s = tbl.%s end", name, name, name, name)
			}
		}
		w.WriteLine("\treturn self, nil\nend")
	}

	// And a helper to initialize an existing object
	{
		w.WriteLinef(`
_%s.%s._initializeFromTable = function(self, tbl)
	if tbl == nil then
		return nil
	end
	if type(tbl) ~= "table" then
		return builtins.create_error("cannot initialize struct from non-table")
	end
	local obj, err
		`, pkgName, s.Name.Name)

		// For each field, add an initializer
		obj := p.identObject(s.Name)
		typ := obj.Type().Underlying().(*types.Struct)
		for i := 0; i < typ.NumFields(); i++ {
			// Get the raw field type
			f := typ.Field(i)
			fType := f.Type()
			var named *types.Named
			for {
				if n, ok := fType.(*types.Named); named == nil && ok {
					named = n
				}

				if ptr, ok := fType.Underlying().(*types.Pointer); ok {
					fType = ptr.Elem()
				} else {
					break
				}
			}

			name := computeFieldName(f.Name(), typ.Tag(i))
			switch fType.Underlying().(type) {
			case *types.Struct:
				if named != nil {
					w.WriteLinef(`
		obj, err = _%s.%s._createFromTable(tbl.%s)
		if err ~= nil then
			return err
		end
		self.%s = obj
					`, named.Obj().Pkg().Name(), named.Obj().Name(), name, name)
				} else {
					w.WriteLinef("self.%s = tbl.%s", name, name)
				}
			case *types.Interface:
				// do nothing, can't deserialize interface types since we don't know
				// which concrete type to use.
			default:
				w.WriteLinef("\tif type(self.%s) == type(tbl.%s) then self.%s = tbl.%s end", name, name, name, name)
			}
		}
		w.WriteLine("\treturn nil\nend")
	}
}
