package domain

type ServerHTTPConfig struct {
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr string // "" means no body check
	Method        string
}
