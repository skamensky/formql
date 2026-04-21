package frontend

import (
	"strings"

	"github.com/skamensky/formql/pkg/formql/diagnostic"
	"github.com/skamensky/formql/pkg/formql/token"
)

type lexer struct {
	input string
	pos   int
	ch    byte
}

func newLexer(input string) *lexer {
	l := &lexer{input: input}
	if len(input) > 0 {
		l.ch = input[0]
	}
	return l
}

func (l *lexer) advance() {
	l.pos++
	if l.pos >= len(l.input) {
		l.ch = 0
		return
	}
	l.ch = l.input[l.pos]
}

func (l *lexer) peek() byte {
	next := l.pos + 1
	if next >= len(l.input) {
		return 0
	}
	return l.input[next]
}

func (l *lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.advance()
	}
}

func (l *lexer) skipLineComment() {
	l.advance()
	l.advance()
	for l.ch != 0 && l.ch != '\n' {
		l.advance()
	}
	if l.ch == '\n' {
		l.advance()
	}
}

func (l *lexer) skipBlockComment() error {
	l.advance()
	l.advance()
	for l.ch != 0 {
		if l.ch == '*' && l.peek() == '/' {
			l.advance()
			l.advance()
			return nil
		}
		l.advance()
	}
	return diagnostic.New("lexer", "unterminated block comment", l.pos)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func (l *lexer) readNumber() token.Token {
	start := l.pos
	seenDot := false
	for l.ch != 0 {
		if l.ch == '.' {
			if seenDot {
				break
			}
			seenDot = true
			l.advance()
			continue
		}
		if !isDigit(l.ch) {
			break
		}
		l.advance()
	}
	return token.Token{
		Type:     token.NUMBER,
		Literal:  l.input[start:l.pos],
		Position: start,
	}
}

func (l *lexer) readIdentifier() token.Token {
	start := l.pos
	for l.ch != 0 && isIdentPart(l.ch) {
		l.advance()
	}
	return token.Token{
		Type:     token.IDENT,
		Literal:  l.input[start:l.pos],
		Position: start,
	}
}

func (l *lexer) readString() (token.Token, error) {
	start := l.pos
	l.advance()
	var b strings.Builder

	for l.ch != 0 && l.ch != '"' {
		if l.ch == '\\' && l.peek() == '"' {
			l.advance()
		}
		b.WriteByte(l.ch)
		l.advance()
	}

	if l.ch == 0 {
		return token.Token{}, diagnostic.New("lexer", "unterminated string literal", start)
	}

	l.advance()
	return token.Token{
		Type:     token.STRING,
		Literal:  b.String(),
		Position: start,
	}, nil
}

func (l *lexer) nextToken() (token.Token, error) {
	for l.ch != 0 {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			l.skipWhitespace()
			continue
		}

		if l.ch == '/' && l.peek() == '/' {
			l.skipLineComment()
			continue
		}

		if l.ch == '/' && l.peek() == '*' {
			if err := l.skipBlockComment(); err != nil {
				return token.Token{}, err
			}
			continue
		}

		if isDigit(l.ch) {
			return l.readNumber(), nil
		}

		if isIdentStart(l.ch) {
			return l.readIdentifier(), nil
		}

		if l.ch == '"' {
			return l.readString()
		}

		current := l.pos
		switch l.ch {
		case '+':
			l.advance()
			return token.Token{Type: token.PLUS, Literal: "+", Position: current}, nil
		case '-':
			l.advance()
			return token.Token{Type: token.MINUS, Literal: "-", Position: current}, nil
		case '*':
			l.advance()
			return token.Token{Type: token.MULTIPLY, Literal: "*", Position: current}, nil
		case '/':
			l.advance()
			return token.Token{Type: token.DIVIDE, Literal: "/", Position: current}, nil
		case '&':
			l.advance()
			return token.Token{Type: token.AMPERSAND, Literal: "&", Position: current}, nil
		case '(':
			l.advance()
			return token.Token{Type: token.LPAREN, Literal: "(", Position: current}, nil
		case ')':
			l.advance()
			return token.Token{Type: token.RPAREN, Literal: ")", Position: current}, nil
		case ',':
			l.advance()
			return token.Token{Type: token.COMMA, Literal: ",", Position: current}, nil
		case '.':
			l.advance()
			return token.Token{Type: token.DOT, Literal: ".", Position: current}, nil
		case '>':
			l.advance()
			if l.ch == '=' {
				l.advance()
				return token.Token{Type: token.GTE, Literal: ">=", Position: current}, nil
			}
			return token.Token{Type: token.GT, Literal: ">", Position: current}, nil
		case '<':
			l.advance()
			if l.ch == '=' {
				l.advance()
				return token.Token{Type: token.LTE, Literal: "<=", Position: current}, nil
			}
			if l.ch == '>' {
				l.advance()
				return token.Token{Type: token.NEQ, Literal: "<>", Position: current}, nil
			}
			return token.Token{Type: token.LT, Literal: "<", Position: current}, nil
		case '=':
			l.advance()
			return token.Token{Type: token.EQ, Literal: "=", Position: current}, nil
		case '!':
			l.advance()
			if l.ch == '=' {
				l.advance()
				return token.Token{Type: token.NEQ, Literal: "!=", Position: current}, nil
			}
			return token.Token{}, diagnostic.New("lexer", "unexpected '!': did you mean '!='?", current)
		default:
			return token.Token{}, diagnostic.New("lexer", "unexpected character "+string(l.ch), current)
		}
	}

	return token.Token{Type: token.EOF, Position: l.pos}, nil
}
