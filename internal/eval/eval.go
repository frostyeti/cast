package eval

import (
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
)

type token struct {
	value string
	expr  bool
}

func EvalTemplate(template string, scope map[string]interface{}) (string, error) {
	tokens := []token{}
	sb := strings.Builder{}
	inExpr := false

	l := len(template)

	for i := 0; i < len(template); i++ {
		c := template[i]
		if !inExpr && c == '$' && i+2 < l && template[i+1] == '{' && template[i+2] == '{' {
			if sb.Len() > 0 {
				tokens = append(tokens, token{value: sb.String(), expr: false})
				sb.Reset()
			}
			inExpr = true
			i += 2
			continue
		}

		if inExpr && c == '}' && i+1 < l && template[i+1] == '}' {
			if sb.Len() > 0 {
				tokens = append(tokens, token{value: sb.String(), expr: true})
				sb.Reset()
			}
			inExpr = false
			i += 1
			continue
		}

		sb.WriteByte(c)
	}

	if sb.Len() > 0 {
		tokens = append(tokens, token{value: sb.String(), expr: inExpr})
	}

	result := strings.Builder{}

	for _, tok := range tokens {
		if tok.expr {
			evaled, err := Eval(tok.value, scope)
			if err != nil {
				return "", err
			}

			result.WriteString(strings.TrimSpace(evaled.(string)))
		} else {
			result.WriteString(tok.value)
		}
	}

	return result.String(), nil
}

func Eval(template string, scope map[string]interface{}) (any, error) {
	block, err := expr.Compile(template)
	if err != nil {
		return "", err
	}

	output, err := expr.Run(block, scope)
	if err != nil {
		return "", err
	}

	return output, nil
}

func EvalAsString(template string, scope map[string]interface{}) (string, error) {
	output, err := Eval(template, scope)
	if err != nil {
		return "", err
	}

	return output.(string), nil
}

func EvalAsInt(template string, scope map[string]interface{}) (int, error) {
	output, err := Eval(template, scope)
	if err != nil {
		return 0, err
	}

	if val, ok := output.(int); ok {
		return val, nil
	}

	if val, ok := output.(float64); ok {
		return int(val), nil
	}

	if val, ok := output.(string); ok {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}

	return 0, nil
}
