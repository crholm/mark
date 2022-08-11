package query

import (
	"errors"
	"reflect"
	"testing"
)

func Test_andOperand(t *testing.T) {
	type args struct {
		tokens []string
	}
	tests := []struct {
		name    string
		args    args
		want    *expr
		want1   []string
		wantErr bool
	}{
		{
			name:    "single string operand before EOF",
			args:    args{[]string{"alice", EOF}},
			want:    &expr{q: "alice"},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "parenthesized operand before EOF",
			args: args{[]string{LPAREN, "alice", OR, "bob", RPAREN, EOF}},
			want: &expr{
				left:  &expr{q: "alice"},
				right: &expr{q: "bob"},
				op:    OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name:    "unclosed parenthesized operand before EOF",
			args:    args{[]string{LPAREN, "alice", OR, "bob", EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name:    "missing operand before EOF",
			args:    args{[]string{EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name:    "RPAREN instead of operand",
			args:    args{[]string{RPAREN, EOF}},
			want:    nil,
			want1:   []string{RPAREN, EOF},
			wantErr: true,
		},
		{
			name:    "OR instead of operand",
			args:    args{[]string{OR, "eve", EOF}},
			want:    nil,
			want1:   []string{OR, "eve", EOF},
			wantErr: true,
		},
		{
			name:    "AND instead of operand",
			args:    args{[]string{AND, "eve", EOF}},
			want:    nil,
			want1:   []string{AND, "eve", EOF},
			wantErr: true,
		},
		{
			name:    "lone EOF not allowed here",
			args:    args{[]string{EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := andOperand(tt.args.tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("andOperand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if eq, err := semantEq(got, tt.want); !eq || err != nil {
				t.Errorf("tree() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("andOperand() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_expression(t *testing.T) {
	type args struct {
		tokens []string
	}
	tests := []struct {
		name    string
		args    args
		want    *expr
		want1   []string
		wantErr bool
	}{
		{
			name:    "single string query",
			args:    args{[]string{"alice:*", EOF}},
			want:    &expr{q: "alice:*"},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "AND query",
			args: args{[]string{"alice:*", AND, "bob:*", EOF}},
			want: &expr{
				left:  &expr{q: "alice:*"},
				right: &expr{q: "bob:*"},
				op:    AND,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "parenthesized AND query",
			args: args{[]string{LPAREN, "alice:*", AND, "bob:*", RPAREN, EOF}},
			want: &expr{
				left:  &expr{q: "alice:*"},
				right: &expr{q: "bob:*"},
				op:    AND,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "OR query",
			args: args{[]string{"alice:*", OR, "bob:*", EOF}},
			want: &expr{
				left:  &expr{q: "alice:*"},
				right: &expr{q: "bob:*"},
				op:    OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "parenthesized OR query",
			args: args{[]string{LPAREN, "alice:*", OR, "bob:*", RPAREN, EOF}},
			want: &expr{
				left:  &expr{q: "alice:*"},
				right: &expr{q: "bob:*"},
				op:    OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "AND query order precedence",
			args: args{[]string{"alice", AND, "bob", AND, "eve", AND, "filippa", EOF}},
			want: &expr{
				left: &expr{
					left: &expr{
						left:  &expr{q: "alice"},
						right: &expr{q: "bob"},
						op:    AND,
					},
					right: &expr{q: "eve"},
					op:    AND,
				},
				right: &expr{q: "filippa"},
				op:    AND,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "AND precedence",
			args: args{[]string{"alice", OR, "bob", AND, "eve", OR, "filippa", EOF}},
			want: &expr{
				left: &expr{
					left: &expr{q: "alice"},
					right: &expr{
						left:  &expr{q: "bob"},
						right: &expr{q: "eve"},
						op:    AND,
					},
					op: OR,
				},
				right: &expr{q: "filippa"},
				op:    OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "parenthesis precedence",
			args: args{[]string{"alice", OR, LPAREN, "bob", OR, LPAREN, "eve", OR, "filippa", RPAREN, RPAREN, EOF}},
			want: &expr{
				left: &expr{q: "alice"},
				right: &expr{
					left: &expr{q: "bob"},
					right: &expr{
						left:  &expr{q: "eve"},
						right: &expr{q: "filippa"},
						op:    OR,
					},
					op: OR,
				},
				op: OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name:    "lone EOF not allowed here",
			args:    args{[]string{EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name:    "unclosed parenthesized expression",
			args:    args{[]string{LPAREN, "alice", AND, "bob", EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name: "double operator",
			args: args{[]string{LPAREN, "alice", AND, AND, "bob", EOF}},
			want: nil,

			want1:   []string{AND, "bob", EOF},
			wantErr: true,
		},
		{
			name: "missing operand after operator",
			args: args{[]string{LPAREN, "alice", AND, EOF}},
			want: nil,

			want1:   []string{EOF},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := expression(tt.args.tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("expression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if eq, err := semantEq(got, tt.want); !eq || err != nil {
				t.Errorf("tree() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("expression() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_nextToken(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name      string
		args      args
		wantToken string
		wantRem   string
	}{
		{
			name:      "empty query",
			args:      args{""},
			wantToken: EOF,
			wantRem:   "",
		},
		{
			name:      "LPAREN before string",
			args:      args{"(alice:*)"},
			wantToken: LPAREN,
			wantRem:   "alice:*)",
		},
		{
			name:      "RPAREN before AND",
			args:      args{")&bob"},
			wantToken: RPAREN,
			wantRem:   "&bob",
		},
		{
			name:      "string before space",
			args:      args{"alice & bob"},
			wantToken: "alice",
			wantRem:   " & bob",
		},
		{
			name:      "AND before string",
			args:      args{"&bob"},
			wantToken: AND,
			wantRem:   "bob",
		},
		{
			name:      "OR before string",
			args:      args{"|bob"},
			wantToken: OR,
			wantRem:   "bob",
		},
		{
			name:      "space before string",
			args:      args{" alice"},
			wantToken: "alice",
			wantRem:   "",
		},
		{
			name:      "space only",
			args:      args{"   "},
			wantToken: EOF,
			wantRem:   "",
		},
		{
			name:      "prefix string before RPAREN",
			args:      args{"alice:*) | bob"},
			wantToken: "alice:*",
			wantRem:   ") | bob",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotRem := nextToken(tt.args.s)
			if gotToken != tt.wantToken {
				t.Errorf("nextToken() gotToken = %v, want %v", gotToken, tt.wantToken)
			}
			if gotRem != tt.wantRem {
				t.Errorf("nextToken() gotRem = %v, want %v", gotRem, tt.wantRem)
			}
		})
	}
}

func Test_orOperand(t *testing.T) {
	type args struct {
		tokens []string
	}
	tests := []struct {
		name    string
		args    args
		want    *expr
		want1   []string
		wantErr bool
	}{
		{
			name:    "single string operand before EOF",
			args:    args{[]string{"alice", EOF}},
			want:    &expr{q: "alice"},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name:    "AND operator rhs precedence",
			args:    args{[]string{"alice", OR, "bob", AND, "eve", EOF}},
			want:    &expr{q: "alice"},
			want1:   []string{OR, "bob", AND, "eve", EOF},
			wantErr: false,
		},
		{
			name: "AND operator lhs precedence",
			args: args{[]string{"alice", AND, "bob", OR, "eve", EOF}},
			want: &expr{
				left:  &expr{q: "alice"},
				right: &expr{q: "bob"},
				op:    AND,
			},
			want1:   []string{OR, "eve", EOF},
			wantErr: false,
		},
		{
			name: "AND operator order precedence",
			args: args{[]string{"alice", AND, "bob", AND, "eve", EOF}},
			want: &expr{
				left:  &expr{left: &expr{q: "alice"}, right: &expr{q: "bob"}, op: AND},
				right: &expr{q: "eve"},
				op:    AND,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name: "parenthesized operand before EOF",
			args: args{[]string{LPAREN, "alice", OR, "bob", RPAREN, EOF}},
			want: &expr{
				left:  &expr{q: "alice"},
				right: &expr{q: "bob"},
				op:    OR,
			},
			want1:   []string{EOF},
			wantErr: false,
		},
		{
			name:    "unclosed parenthesized operand before EOF",
			args:    args{[]string{LPAREN, "alice", OR, "bob", EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name:    "missing operand before EOF",
			args:    args{[]string{EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
		{
			name:    "RPAREN instead of operand",
			args:    args{[]string{RPAREN, EOF}},
			want:    nil,
			want1:   []string{RPAREN, EOF},
			wantErr: true,
		},
		{
			name:    "OR instead of operand",
			args:    args{[]string{OR, "eve", EOF}},
			want:    nil,
			want1:   []string{OR, "eve", EOF},
			wantErr: true,
		},
		{
			name:    "AND instead of operand",
			args:    args{[]string{AND, "eve", EOF}},
			want:    nil,
			want1:   []string{AND, "eve", EOF},
			wantErr: true,
		},
		{
			name:    "lone EOF not allowed here",
			args:    args{[]string{EOF}},
			want:    nil,
			want1:   []string{EOF},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := orOperand(tt.args.tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("orOperand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if eq, err := semantEq(got, tt.want); !eq || err != nil {
				t.Errorf("tree() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("orOperand() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_tokenize(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		wantTokens []string
	}{
		{
			name:       "empty query",
			args:       args{""},
			wantTokens: []string{EOF},
		},
		{
			name:       "single word query",
			args:       args{"alice"},
			wantTokens: []string{"alice", EOF},
		},
		{
			name:       "parenthesized query",
			args:       args{"(alice)"},
			wantTokens: []string{LPAREN, "alice", RPAREN, EOF},
		},
		{
			name:       "and-query",
			args:       args{"alice&bob"},
			wantTokens: []string{"alice", AND, "bob", EOF},
		},
		{
			name:       "or-query",
			args:       args{"alice|bob"},
			wantTokens: []string{"alice", OR, "bob", EOF},
		},
		{
			name:       "spaced query",
			args:       args{" (alice | bob)  &  eve "},
			wantTokens: []string{LPAREN, "alice", OR, "bob", RPAREN, AND, "eve", EOF},
		},
		{
			name:       "prefix match query",
			args:       args{"( alice:* | bob:* ) & eve:* "},
			wantTokens: []string{LPAREN, "alice:*", OR, "bob:*", RPAREN, AND, "eve:*", EOF},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotTokens := tokenize(tt.args.s); !reflect.DeepEqual(gotTokens, tt.wantTokens) {
				t.Errorf("tokenize() = %v, want %v", gotTokens, tt.wantTokens)
			}
		})
	}
}

func Test_tree(t *testing.T) {
	type args struct {
		tokens []string
	}
	tests := []struct {
		name    string
		args    args
		want    *expr
		wantErr bool
	}{
		{
			name:    "lone EOF ok",
			args:    args{[]string{EOF}},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "missing operator",
			args:    args{[]string{"alice", "bob", EOF}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unclosed parenthesized expression",
			args:    args{[]string{LPAREN, "alice", AND, "bob", EOF}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "double operator",
			args:    args{[]string{LPAREN, "alice", AND, AND, "bob", EOF}},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing operand after operator",
			args:    args{[]string{LPAREN, "alice", AND, EOF}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tree(tt.args.tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("tree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if eq, err := semantEq(got, tt.want); !eq || err != nil {
				t.Errorf("tree() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func semantEq(e1 *expr, e2 *expr) (bool, error) {
	literals := make(map[string]bool)
	for _, s := range leafOperands(e1) {
		literals[s] = false
	}
	for _, s := range leafOperands(e2) {
		literals[s] = false
	}

	var queue []string
	for literal := range literals {
		queue = append(queue, literal)
	}

	var brute func(queue []string) (bool, error)
	brute = func(queue []string) (bool, error) {
		if len(queue) == 0 {
			return true, nil
		}
		head, tail := queue[0], queue[1:]

		literals[head] = false
		b1, err := boolEval(e1, literals)
		if err != nil {
			return false, err
		}
		b2, err := boolEval(e2, literals)
		if err != nil {
			return false, err
		}
		if b1 != b2 {
			return false, nil
		}
		if b, err := brute(tail); !b || err != nil {
			return false, err
		}

		literals[head] = true
		b1, err = boolEval(e1, literals)
		if err != nil {
			return false, err
		}
		b2, err = boolEval(e2, literals)
		if err != nil {
			return false, err
		}
		if b1 != b2 {
			return false, nil
		}
		if b, err := brute(tail); !b || err != nil {
			return false, err
		}

		return true, nil
	}
	return brute(queue)
}

func boolEval(e *expr, literals map[string]bool) (bool, error) {
	if e == nil {
		return false, errors.New("can't evaluate nil expr")
	}
	if len(e.q) > 0 {
		b, ok := literals[e.q]
		if !ok {
			return false, errors.New("unknown literal: " + e.q)
		}
		return b, nil
	}

	l, err := boolEval(e.left, literals)
	if err != nil {
		return false, err
	}

	r, err := boolEval(e.right, literals)
	if err != nil {
		return false, err
	}

	if e.op == AND {
		return l && r, nil
	}

	if e.op == OR {
		return l || r, nil
	}

	return false, errors.New("unknown operand: " + e.op)
}

func leafOperands(e *expr) []string {
	if e == nil {
		return nil
	}
	if len(e.q) > 0 {
		return []string{e.q}
	}

	return append(leafOperands(e.left), leafOperands(e.right)...)
}
