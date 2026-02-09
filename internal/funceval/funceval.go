package funceval

import (
	"math"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Evaluator wraps a compiled expression.
type Evaluator struct {
	program *vm.Program
	env     map[string]interface{}
}

// Map of standard math functions to expose to expr
var mathEnv = map[string]interface{}{
	"sin":  math.Sin,
	"cos":  math.Cos,
	"tan":  math.Tan,
	"exp":  math.Exp,
	"log":  math.Log,
	"sqrt": math.Sqrt,
	"pow":  math.Pow,
	"abs":  math.Abs,
	"pi":   math.Pi,
	"e":    math.E,
}

// Compile parses and compiles an expression.
// The expression can use 'x' as a variable and common math functions.
func Compile(expression string) (*Evaluator, error) {
	// Create a combined environment for compilation
	combinedEnv := make(map[string]interface{})
	for k, v := range mathEnv {
		combinedEnv[k] = v
	}
	combinedEnv["x"] = 0.0 // Placeholder for type inference

	program, err := expr.Compile(expression, expr.Env(combinedEnv))
	if err != nil {
		return nil, err
	}

	return &Evaluator{
		program: program,
		env:     combinedEnv,
	}, nil
}

// Eval evaluates the compiled expression for a given x.
func (e *Evaluator) Eval(x float64) (float64, error) {
	e.env["x"] = x
	output, err := expr.Run(e.program, e.env)
	if err != nil {
		return 0, err
	}

	// Cast the result to float64. expr might return int if the result is an integer.
	switch v := output.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, nil // Or handle error
	}
}
