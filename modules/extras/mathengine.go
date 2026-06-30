package extras

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	modules "main/modules"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type mathToken struct {
	kind  int
	value float64
	op    byte
}

const (
	mtNum = iota
	mtOp
	mtLParen
	mtRParen
)

func mathPrec(op byte) int {
	switch op {
	case '+', '-':
		return 1
	case '*', '/':
		return 2
	case '^':
		return 3
	}
	return 0
}

func mathRightAssoc(op byte) bool {
	return op == '^'
}

func mathTokenize(s string) ([]mathToken, error) {
	tokens := []mathToken{}
	i := 0
	prevWasOp := true
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if (c >= '0' && c <= '9') || c == '.' {
			j := i
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			v, err := strconv.ParseFloat(s[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("bad number: %s", s[i:j])
			}
			tokens = append(tokens, mathToken{kind: mtNum, value: v})
			i = j
			prevWasOp = false
			continue
		}
		if c == '(' {
			tokens = append(tokens, mathToken{kind: mtLParen})
			i++
			prevWasOp = true
			continue
		}
		if c == ')' {
			tokens = append(tokens, mathToken{kind: mtRParen})
			i++
			prevWasOp = false
			continue
		}
		if c == '+' || c == '-' || c == '*' || c == '/' || c == '^' {
			if (c == '+' || c == '-') && prevWasOp {
				j := i + 1
				for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
					j++
				}
				if j == i+1 {
					return nil, fmt.Errorf("unexpected '%c'", c)
				}
				v, err := strconv.ParseFloat(s[i:j], 64)
				if err != nil {
					return nil, fmt.Errorf("bad number: %s", s[i:j])
				}
				tokens = append(tokens, mathToken{kind: mtNum, value: v})
				i = j
				prevWasOp = false
				continue
			}
			tokens = append(tokens, mathToken{kind: mtOp, op: c})
			i++
			prevWasOp = true
			continue
		}
		return nil, fmt.Errorf("invalid char '%c'", c)
	}
	return tokens, nil
}

func mathShuntingYard(tokens []mathToken) ([]mathToken, error) {
	out := []mathToken{}
	stack := []mathToken{}
	for _, t := range tokens {
		switch t.kind {
		case mtNum:
			out = append(out, t)
		case mtOp:
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if top.kind != mtOp {
					break
				}
				if mathPrec(top.op) > mathPrec(t.op) || (mathPrec(top.op) == mathPrec(t.op) && !mathRightAssoc(t.op)) {
					out = append(out, top)
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			stack = append(stack, t)
		case mtLParen:
			stack = append(stack, t)
		case mtRParen:
			matched := false
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if top.kind == mtLParen {
					matched = true
					break
				}
				out = append(out, top)
			}
			if !matched {
				return nil, fmt.Errorf("mismatched parentheses")
			}
		}
	}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if top.kind == mtLParen || top.kind == mtRParen {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		out = append(out, top)
	}
	return out, nil
}

func mathEvalRPN(rpn []mathToken) (float64, error) {
	stack := []float64{}
	for _, t := range rpn {
		if t.kind == mtNum {
			stack = append(stack, t.value)
			continue
		}
		if len(stack) < 2 {
			return 0, fmt.Errorf("malformed expression")
		}
		b := stack[len(stack)-1]
		a := stack[len(stack)-2]
		stack = stack[:len(stack)-2]
		var r float64
		switch t.op {
		case '+':
			r = a + b
		case '-':
			r = a - b
		case '*':
			r = a * b
		case '/':
			if b == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			r = a / b
		case '^':
			r = math.Pow(a, b)
		default:
			return 0, fmt.Errorf("unknown op %c", t.op)
		}
		if math.IsNaN(r) || math.IsInf(r, 0) {
			return 0, fmt.Errorf("result overflow")
		}
		stack = append(stack, r)
	}
	if len(stack) != 1 {
		return 0, fmt.Errorf("malformed expression")
	}
	return stack[0], nil
}

func mathEvaluate(expr string) (float64, error) {
	tokens, err := mathTokenize(expr)
	if err != nil {
		return 0, err
	}
	if len(tokens) == 0 {
		return 0, fmt.Errorf("empty expression")
	}
	rpn, err := mathShuntingYard(tokens)
	if err != nil {
		return 0, err
	}
	return mathEvalRPN(rpn)
}

func mathFormat(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func CalcHandler(m *tg.NewMessage) error {
	expr := strings.TrimSpace(m.Args())
	if expr == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/calc &lt;expression&gt;</code>\n\n<b>Example:</b>\n<code>/calc 2 + 3 * (4 - 1) ^ 2</code>\n\nSupports: <code>+ - * / ^ ( )</code>")
		return err
	}
	result, err := mathEvaluate(expr)
	if err != nil {
		_, e := m.Reply("error: " + html.EscapeString(err.Error()))
		return e
	}
	reply := fmt.Sprintf("<code>%s</code> = <b><code>%s</code></b>", html.EscapeString(expr), mathFormat(result))
	_, err = m.Reply(reply)
	return err
}

func eqParseSide(side string, varName byte) (float64, float64, error) {
	side = strings.ReplaceAll(side, " ", "")
	if side == "" {
		return 0, 0, fmt.Errorf("empty side")
	}
	if side[0] != '+' && side[0] != '-' {
		side = "+" + side
	}
	var coefSum, constSum float64
	i := 0
	for i < len(side) {
		sign := 1.0
		switch side[i] {
		case '+':
			i++
		case '-':
			sign = -1.0
			i++
		default:
			return 0, 0, fmt.Errorf("unexpected '%c'", side[i])
		}
		j := i
		hasDigit := false
		for j < len(side) && ((side[j] >= '0' && side[j] <= '9') || side[j] == '.') {
			hasDigit = true
			j++
		}
		numStr := side[i:j]
		var num float64
		if hasDigit {
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("bad number: %s", numStr)
			}
			num = n
		} else {
			num = 1.0
		}
		if j < len(side) && side[j] == varName {
			coefSum += sign * num
			j++
		} else {
			if !hasDigit {
				return 0, 0, fmt.Errorf("expected number or variable at %d", i)
			}
			constSum += sign * num
		}
		i = j
	}
	return coefSum, constSum, nil
}

func eqDetectVar(s string) (byte, error) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return c, nil
		}
	}
	return 0, fmt.Errorf("no variable found")
}

func EqSolveHandler(m *tg.NewMessage) error {
	expr := strings.TrimSpace(m.Args())
	if expr == "" {
		_, err := m.Reply("<b>Usage:</b> <code>/eqsolve &lt;expr=expr&gt;</code>\n\n<b>Example:</b>\n<code>/eqsolve 2x+3=11</code>\n<code>/eqsolve 5y - 7 = 2y + 8</code>")
		return err
	}
	parts := strings.Split(expr, "=")
	if len(parts) != 2 {
		_, e := m.Reply("error: equation must contain exactly one '='")
		return e
	}
	lhs := strings.TrimSpace(parts[0])
	rhs := strings.TrimSpace(parts[1])
	if lhs == "" || rhs == "" {
		_, e := m.Reply("error: both sides must be non-empty")
		return e
	}
	varName, err := eqDetectVar(lhs + rhs)
	if err != nil {
		_, e := m.Reply("error: " + html.EscapeString(err.Error()))
		return e
	}
	lCoef, lConst, err := eqParseSide(lhs, varName)
	if err != nil {
		_, e := m.Reply("error parsing lhs: " + html.EscapeString(err.Error()))
		return e
	}
	rCoef, rConst, err := eqParseSide(rhs, varName)
	if err != nil {
		_, e := m.Reply("error parsing rhs: " + html.EscapeString(err.Error()))
		return e
	}
	coef := lCoef - rCoef
	constant := rConst - lConst
	if coef == 0 {
		if constant == 0 {
			_, e := m.Reply("<code>" + html.EscapeString(expr) + "</code>\n\n<b>infinitely many solutions</b>")
			return e
		}
		_, e := m.Reply("<code>" + html.EscapeString(expr) + "</code>\n\n<b>no solution</b>")
		return e
	}
	result := constant / coef
	reply := fmt.Sprintf("<code>%s</code>\n\n<b>%c = %s</b>", html.EscapeString(expr), varName, mathFormat(result))
	_, err = m.Reply(reply)
	return err
}

var unitLength = map[string]float64{
	"mm": 0.001,
	"cm": 0.01,
	"m":  1.0,
	"km": 1000.0,
	"in": 0.0254,
	"ft": 0.3048,
	"yd": 0.9144,
	"mi": 1609.344,
}

var unitMass = map[string]float64{
	"mg": 0.001,
	"g":  1.0,
	"kg": 1000.0,
	"t":  1000000.0,
	"oz": 28.3495,
	"lb": 453.592,
	"st": 6350.29,
}

var unitTemp = map[string]bool{
	"c": true,
	"f": true,
	"k": true,
}

func unitConvertTemp(value float64, from, to string) (float64, error) {
	var kelvin float64
	switch from {
	case "c":
		kelvin = value + 273.15
	case "f":
		kelvin = (value-32)*5/9 + 273.15
	case "k":
		kelvin = value
	default:
		return 0, fmt.Errorf("unknown temp unit: %s", from)
	}
	switch to {
	case "c":
		return kelvin - 273.15, nil
	case "f":
		return (kelvin-273.15)*9/5 + 32, nil
	case "k":
		return kelvin, nil
	}
	return 0, fmt.Errorf("unknown temp unit: %s", to)
}

func unitConvertScale(value float64, from, to string, table map[string]float64) (float64, error) {
	fromFactor, okFrom := table[from]
	toFactor, okTo := table[to]
	if !okFrom || !okTo {
		return 0, fmt.Errorf("unknown unit")
	}
	return value * fromFactor / toFactor, nil
}

func UnitConvHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 3 {
		_, err := m.Reply("<b>Usage:</b> <code>/unitconv &lt;n&gt; &lt;from&gt; &lt;to&gt;</code>\n\n<b>Examples:</b>\n<code>/unitconv 10 km mi</code>\n<code>/unitconv 150 lb kg</code>\n<code>/unitconv 100 f c</code>\n\n<b>Supported:</b>\n<i>length:</i> <code>mm cm m km in ft yd mi</code>\n<i>mass:</i>   <code>mg g kg t oz lb st</code>\n<i>temp:</i>   <code>c f k</code>")
		return err
	}
	value, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		_, e := m.Reply("error: invalid number <code>" + html.EscapeString(args[0]) + "</code>")
		return e
	}
	from := strings.ToLower(args[1])
	to := strings.ToLower(args[2])

	var result float64
	var category string

	if unitTemp[from] && unitTemp[to] {
		r, err := unitConvertTemp(value, from, to)
		if err != nil {
			_, e := m.Reply("error: " + html.EscapeString(err.Error()))
			return e
		}
		result = r
		category = "temperature"
	} else if _, okF := unitLength[from]; okF {
		if _, okT := unitLength[to]; !okT {
			_, e := m.Reply("error: unit category mismatch")
			return e
		}
		r, err := unitConvertScale(value, from, to, unitLength)
		if err != nil {
			_, e := m.Reply("error: " + html.EscapeString(err.Error()))
			return e
		}
		result = r
		category = "length"
	} else if _, okF := unitMass[from]; okF {
		if _, okT := unitMass[to]; !okT {
			_, e := m.Reply("error: unit category mismatch")
			return e
		}
		r, err := unitConvertScale(value, from, to, unitMass)
		if err != nil {
			_, e := m.Reply("error: " + html.EscapeString(err.Error()))
			return e
		}
		result = r
		category = "mass"
	} else {
		_, e := m.Reply("error: unknown unit <code>" + html.EscapeString(from) + "</code>")
		return e
	}

	reply := fmt.Sprintf("<b>Unit Conversion</b> <i>(%s)</i>\n\n<code>%s %s</code> = <b><code>%s %s</code></b>",
		category,
		mathFormat(value), html.EscapeString(from),
		mathFormat(result), html.EscapeString(to))
	_, err = m.Reply(reply)
	return err
}

func registerMathEngineHandlers() {
	c := modules.Client
	c.On("cmd:calc", CalcHandler)
	c.On("cmd:eqsolve", EqSolveHandler)
	c.On("cmd:unitconv", UnitConvHandler)
}

func init() {
	modules.QueueHandlerRegistration(registerMathEngineHandlers)
}
