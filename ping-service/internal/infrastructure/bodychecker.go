package infrastructure

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/samber/do/v2"
)

type BodyChecker struct{}

func RegisterBodyChecker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*BodyChecker, error) {
		return &BodyChecker{}, nil
	})
}

func (c *BodyChecker) Check(body string, expression string) (bool, error) {

	env := map[string]any{
		"body": body,
	}

	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return false, fmt.Errorf("compile body check expr: %w", err)
	}

	out, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("run body check expr: %w", err)
	}

	ok, _ := out.(bool)
	return ok, nil
}
