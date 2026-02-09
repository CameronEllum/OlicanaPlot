package funceval

import (
	"math"
	"testing"
)

func TestCompileAndEval(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		x          float64
		expected   float64
		tolerance  float64
	}{
		{"simple linear", "x * 2", 5, 10, 0.0001},
		{"sine wave", "sin(x)", math.Pi / 2, 1, 0.0001},
		{"damped wave", "exp(-0.1 * x) * sin(x)", math.Pi / 2, math.Exp(-0.1 * math.Pi / 2), 0.0001},
		{"power", "x ^ 2", 3, 9, 0.0001},
		{"constants", "x + pi", 1, 1 + math.Pi, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval, err := Compile(tt.expression)
			if err != nil {
				t.Fatalf("Compile failed: %v", err)
			}
			got, err := eval.Eval(tt.x)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}
			if math.Abs(got-tt.expected) > tt.tolerance {
				t.Errorf("Eval() = %v, want %v", got, tt.expected)
			}
		})
	}
}
