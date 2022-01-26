// Package colorize use AST analysis to analyze the source
// and colorize the different kinds of literals, like keywords,
// imported packages, etc.
package colorize

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
)

// Print prints to out a syntax highlighted version of the text read from
// reader, between lines startLine and endLine.
func Print(out io.Writer, path string, reader io.Reader, startLine, endLine, arrowLine int, colorEscapes map[Style]string) error {
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	w := &lineWriter{w: out, lineRange: [2]int{startLine, endLine}, arrowLine: arrowLine, colorEscapes: colorEscapes}

	if filepath.Ext(path) != ".go" {
		w.Write(NormalStyle, buf, true)
		return nil
	}

	var fset token.FileSet
	f, err := parser.ParseFile(&fset, path, buf, parser.ParseComments)
	if err != nil {
		w.Write(NormalStyle, buf, true)
		return nil
	}

	var base int

	fset.Iterate(func(file *token.File) bool {
		base = file.Base()
		return false
	})

	type colorTok struct {
		tok        token.Token // the token type or ILLEGAL for keywords
		start, end int         // start and end positions of the token
	}

	toks := []colorTok{}

	emit := func(tok token.Token, start, end token.Pos) {
		if _, ok := tokenToStyle[tok]; !ok {
			return
		}
		start -= token.Pos(base)
		if end == token.NoPos {
			// end == token.NoPos it's a keyword and we have to find where it ends by looking at the file
			for end = start; end < token.Pos(len(buf)); end++ {
				if buf[end] < 'a' || buf[end] > 'z' {
					break
				}
			}
		} else {
			end -= token.Pos(base)
		}
		if start < 0 || start >= end || end > token.Pos(len(buf)) {
			// invalid token?
			return
		}
		toks = append(toks, colorTok{tok, int(start), int(end)})
	}

	emit(token.PACKAGE, f.Package, token.NoPos)

	for _, cgrp := range f.Comments {
		for _, cmnt := range cgrp.List {
			emit(token.COMMENT, cmnt.Pos(), cmnt.End())
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch n := n.(type) {
		case *ast.BasicLit:
			emit(n.Kind, n.Pos(), n.End())
			return true
		case *ast.Ident:
			// TODO(aarzilli): builtin functions? basic types?
			return true
		case *ast.IfStmt:
			emit(token.IF, n.If, token.NoPos)
			if n.Else != nil {
				for elsepos := int(n.Body.End()) - base; elsepos < len(buf)-4; elsepos++ {
					if string(buf[elsepos:][:4]) == "else" {
						emit(token.ELSE, token.Pos(elsepos+base), token.Pos(elsepos+base+4))
						break
					}
				}
			}
			return true
		}

		nval := reflect.ValueOf(n)
		if nval.Kind() != reflect.Ptr {
			return true
		}
		nval = nval.Elem()
		if nval.Kind() != reflect.Struct {
			return true
		}

		tokposval := nval.FieldByName("TokPos")
		tokval := nval.FieldByName("Tok")
		if tokposval != (reflect.Value{}) && tokval != (reflect.Value{}) {
			emit(tokval.Interface().(token.Token), tokposval.Interface().(token.Pos), token.NoPos)
		}

		for _, kwname := range []string{"Case", "Begin", "Defer", "Pacakge", "For", "Func", "Go", "Interface", "Map", "Return", "Select", "Struct", "Switch"} {
			kwposval := nval.FieldByName(kwname)
			if kwposval != (reflect.Value{}) {
				kwpos, ok := kwposval.Interface().(token.Pos)
				if ok {
					emit(token.ILLEGAL, kwpos, token.NoPos)
				}
			}
		}

		return true
	})

	sort.Slice(toks, func(i, j int) bool { return toks[i].start < toks[j].start })

	flush := func(start, end int, style Style) {
		if start < end {
			w.Write(style, buf[start:end], end == len(buf))
		}
	}

	cur := 0
	for _, tok := range toks {
		flush(cur, tok.start, NormalStyle)
		flush(tok.start, tok.end, tokenToStyle[tok.tok])
		cur = tok.end
	}
	if cur != len(buf) {
		flush(cur, len(buf), NormalStyle)
	}

	return nil
}
