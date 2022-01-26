package colorize

import "go/token"

// Style describes the style of a chunk of text.
type Style uint8

const (
	NormalStyle Style = iota
	KeywordStyle
	StringStyle
	NumberStyle
	CommentStyle
	LineNoStyle
	ArrowStyle
)

var tokenToStyle = map[token.Token]Style{
	token.ILLEGAL:     KeywordStyle,
	token.COMMENT:     CommentStyle,
	token.INT:         NumberStyle,
	token.FLOAT:       NumberStyle,
	token.IMAG:        NumberStyle,
	token.CHAR:        StringStyle,
	token.STRING:      StringStyle,
	token.BREAK:       KeywordStyle,
	token.CASE:        KeywordStyle,
	token.CHAN:        KeywordStyle,
	token.CONST:       KeywordStyle,
	token.CONTINUE:    KeywordStyle,
	token.DEFAULT:     KeywordStyle,
	token.DEFER:       KeywordStyle,
	token.ELSE:        KeywordStyle,
	token.FALLTHROUGH: KeywordStyle,
	token.FOR:         KeywordStyle,
	token.FUNC:        KeywordStyle,
	token.GO:          KeywordStyle,
	token.GOTO:        KeywordStyle,
	token.IF:          KeywordStyle,
	token.IMPORT:      KeywordStyle,
	token.INTERFACE:   KeywordStyle,
	token.MAP:         KeywordStyle,
	token.PACKAGE:     KeywordStyle,
	token.RANGE:       KeywordStyle,
	token.RETURN:      KeywordStyle,
	token.SELECT:      KeywordStyle,
	token.STRUCT:      KeywordStyle,
	token.SWITCH:      KeywordStyle,
	token.TYPE:        KeywordStyle,
	token.VAR:         KeywordStyle,
}
