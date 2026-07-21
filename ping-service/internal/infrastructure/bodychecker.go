package infrastructure

import (
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/samber/do/v2"
)

type BodyChecker struct{}

func RegisterBodyChecker(i do.Injector) {
	do.Provide(i, func(_ do.Injector) (*BodyChecker, error) {
		return &BodyChecker{}, nil
	})
}

func (c *BodyChecker) Check(body string, expression string) (bool, error) {

	var object any
	if err := json.Unmarshal([]byte(body), &object); err != nil {
		return false, fmt.Errorf("unmarshal body: %w", err)
	}

	program, err := expr.Compile(expression, expr.Env(object))
	if err != nil {
		return false, fmt.Errorf("compile body check expr: %w", err)
	}

	out, err := expr.Run(program, object)
	if err != nil {
		return false, fmt.Errorf("run body check expr: %w", err)
	}

	ok, _ := out.(bool)
	return ok, nil
}
