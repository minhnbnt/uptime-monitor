package dto

type HttpConfig struct {
	Port          int    `json:"port"`
	EndpointPath  string `json:"endpoint_path"`
	ExpectedCode  int    `json:"expected_code"`
	BodyCheckExpr string `json:"body_check_expr"`
}
