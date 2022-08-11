package query

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mark/internal/tsar"
	"sort"
	"strings"
)

const EOF string = ""
const SPACE string = " "

const LPAREN string = "("
const RPAREN string = ")"

const AND string = "&"
const OR string = "|"

type ParseError struct {
	tok []string
	rem []string
	err error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("when parsing %s, after %s: %s",
		strings.TrimSpace(strings.Join(e.tok, " ")),
		strings.TrimSpace(strings.Join(e.tok[:len(e.tok)-len(e.rem)], " ")),
		e.err.Error(),
	)
}

func (e *ParseError) Unwrap() error {
	return e.err
}

func parseErr(tok []string, rem []string, err error) *ParseError {
	return &ParseError{
		tok: tok,
		rem: rem,
		err: err,
	}
}

type SyntaxError struct {
	exp []string
	rem []string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("expected %s, found %s", strings.Join(e.exp, " or "), e.rem[0])
}

func syntaxErr(rem []string, exp ...string) *SyntaxError {
	return &SyntaxError{
		exp: exp,
		rem: rem,
	}
}

type expr struct {
	left  *expr
	right *expr
	op    string
	q     string
}

func tokenize(s string) (tokens []string) {
	var token string
	for ok := true; ok; ok = token != EOF {
		token, s = nextToken(s)
		tokens = append(tokens, token)
	}
	return tokens
}

func nextToken(s string) (token string, rem string) {
	s = strings.TrimSpace(s)

	if len(s) == 0 {
		return EOF, s
	}

	switch s[0:1] {
	case LPAREN, RPAREN, AND, OR:
		return s[0:1], s[1:]
	}

	for i := 0; i < len(s); i++ {
		switch s[i : i+1] {
		case LPAREN, RPAREN, AND, OR, SPACE:
			return s[:i], s[i:]
		}
	}

	return s, ""
}

func sprint(tokens []string) string {
	var b []string
	for _, t := range tokens {
		b = append(b, t)
	}
	return strings.Join(b, " ")
}

func sprintExp(exp *expr) string {
	if exp == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{L: %v, R: %v, op: %s q: %s }", sprintExp(exp.left), sprintExp(exp.right), exp.op, exp.q)
}

func tree(tokens []string) (*expr, error) {
	if len(tokens) == 0 || (len(tokens) == 1 && tokens[0] == EOF) {
		return nil, nil
	}

	e, rem, err := expression(tokens)
	if err == nil && len(rem) == 0 {
		err = errors.New("missing EOF") // should never happen, indicates err in tokenizer or parser
	}
	if err == nil && rem[0] != EOF {
		err = syntaxErr(rem, "operator")
	}

	if err != nil {
		return nil, parseErr(tokens, rem, err)
	}
	return e, nil
}

func expression(tokens []string) (*expr, []string, error) {
	e, tokens, err := orOperand(tokens)
	if err != nil {
		return nil, tokens, err
	}

	for tokens[0] == OR {
		var rExpr *expr
		rExpr, rem, err := orOperand(tokens[1:])
		if err != nil {
			return nil, rem, err
		}
		e = &expr{left: e, right: rExpr, op: OR}
		tokens = rem
	}

	return e, tokens, nil
}

func orOperand(tokens []string) (*expr, []string, error) {
	e, tokens, err := andOperand(tokens)
	if err != nil {
		return nil, tokens, err
	}

	for tokens[0] == AND {
		var rExpr *expr
		rExpr, rem, err := andOperand(tokens[1:])
		if err != nil {
			return nil, rem, err
		}
		e = &expr{left: e, right: rExpr, op: AND}
		tokens = rem
	}

	return e, tokens, nil
}

func andOperand(tokens []string) (*expr, []string, error) {
	switch tokens[0] {
	case LPAREN:
		e, rem, err := expression(tokens[1:])
		if err != nil {
			return nil, rem, err
		}
		if rem[0] == RPAREN {
			return e, rem[1:], nil
		}
		return nil, rem, syntaxErr(rem, "closing parenthesis", "operator")

	case RPAREN, AND, OR, EOF:
		return nil, tokens, syntaxErr(tokens, "expression")

	default:
		e := &expr{q: tokens[0]}
		return e, tokens[1:], nil
	}
}

func eval(rootExp *expr, index *tsar.Index) ([]uint32, error) {
	var do func(exp *expr) (map[uint32]bool, error)
	do = func(e *expr) (map[uint32]bool, error) {
		if len(e.q) > 0 {
			matcher := tsar.MatchEqual
			if strings.HasSuffix(e.q, ":*") {
				e.q = strings.TrimSuffix(e.q, ":*")
				matcher = tsar.MatchPrefix
			}

			entries, err := index.Find(e.q, matcher)
			if err != nil {
				return nil, err
			}

			m := make(map[uint32]bool)
			for _, entry := range entries {
				for _, u := range entry.Pointers {
					m[u] = true
				}
			}
			return m, nil
		}

		a, err := do(e.left)
		if err != nil {
			return nil, err
		}
		b, err := do(e.right)
		if err != nil {
			return nil, err
		}

		switch e.op {
		case AND:
			m := make(map[uint32]bool)
			if len(b) < len(a) {
				a, b = b, a
			}
			for u := range a {
				if _, ok := b[u]; ok {
					m[u] = true
				}
			}
			return m, nil
		case OR:
			m := make(map[uint32]bool)
			for u := range a {
				m[u] = true
			}
			for u := range b {
				m[u] = true
			}
			return m, nil
		}

		return nil, errors.New("eval should not have gotten here")
	}
	m, err := do(rootExp)
	if err != nil {
		return nil, err
	}

	var res []uint32
	for u := range m {
		res = append(res, u)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})

	return res, nil
}

func Query(query string, file io.ReadSeeker, index io.ReadSeeker, limit, offset int) ([]byte, error) {
	query = strings.ToLower(query)
	tokens := tokenize(query)

	exp, err := tree(tokens)
	if err != nil {
		return nil, err
	}

	_, err = index.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	idx, err := tsar.UnmarshalIndexLazyReader(index)
	if err != nil {
		return nil, err
	}

	rows, err := eval(exp, idx)
	if err != nil {
		return nil, err
	}

	var buf []byte
	for i, r := range rows {
		if i < offset {
			continue
		}
		if i >= limit+offset {
			break
		}

		r := int64(r)
		_, err := file.Seek(r, 0)
		if err != nil {
			return nil, err
		}
		reader := bufio.NewReaderSize(file, 512)
		b, err := reader.ReadBytes('\n')
		if err != nil && (err != io.EOF || len(b) == 0) {
			return nil, err
		}
		buf = append(buf, b...)
	}

	return buf, nil
}
