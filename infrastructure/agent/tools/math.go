package tools

import (
	"fmt"
	"strconv"

	"github.com/renesul/ok/domain"
	"strings"
	"unicode"
)

type MathTool struct{}

func (t *MathTool) Name() string        { return "math" }
func (t *MathTool) Description() string { return "evaluates simple mathematical expressions (e.g. 2+3*4)" }
func (t *MathTool) Safety() domain.ToolSafety          { return domain.ToolSafe }

func (t *MathTool) Run(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty expression")
	}

	input = strings.ReplaceAll(input, " ", "")

	// Validate: only digits, operators, dots, parens
	for _, c := range input {
		if !unicode.IsDigit(c) && c != '+' && c != '-' && c != '*' && c != '/' && c != '.' && c != '(' && c != ')' {
			return "", fmt.Errorf("invalid character: %c", c)
		}
	}

	result, err := evalExpr(input)
	if err != nil {
		return "", err
	}

	if result == float64(int64(result)) {
		return strconv.FormatInt(int64(result), 10), nil
	}
	return strconv.FormatFloat(result, 'f', 6, 64), nil
}

// Simple recursive descent parser for +, -, *, /
func evalExpr(s string) (float64, error) {
	p := &parser{input: s}
	result := p.parseExpr()
	if p.err != nil {
		return 0, p.err
	}
	return result, nil
}

type parser struct {
	input string
	pos   int
	err   error
}

func (p *parser) parseExpr() float64 {
	result := p.parseTerm()
	for p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
		op := p.input[p.pos]
		p.pos++
		right := p.parseTerm()
		if op == '+' {
			result += right
		} else {
			result -= right
		}
	}
	return result
}

func (p *parser) parseTerm() float64 {
	result := p.parseFactor()
	for p.pos < len(p.input) && (p.input[p.pos] == '*' || p.input[p.pos] == '/') {
		op := p.input[p.pos]
		p.pos++
		right := p.parseFactor()
		if op == '*' {
			result *= right
		} else {
			if right == 0 {
				p.err = fmt.Errorf("division by zero")
				return 0
			}
			result /= right
		}
	}
	return result
}

func (p *parser) parseFactor() float64 {
	if p.pos >= len(p.input) {
		p.err = fmt.Errorf("incomplete expression")
		return 0
	}

	if p.input[p.pos] == '(' {
		p.pos++
		result := p.parseExpr()
		if p.pos < len(p.input) && p.input[p.pos] == ')' {
			p.pos++
		}
		return result
	}

	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '.') {
		p.pos++
	}

	if start == p.pos {
		p.err = fmt.Errorf("number expected at position %d", p.pos)
		return 0
	}

	val, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		p.err = fmt.Errorf("invalid number: %s", p.input[start:p.pos])
		return 0
	}
	return val
}
