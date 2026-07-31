package domain

type ServerHTTPConfig struct {
	ServerID      uint   `json:"server_id"`
	Port          int    `json:"port"`
	EndpointPath  string `json:"endpoint_path"`
	ExpectedCode  int    `json:"expected_code"`
	BodyCheckExpr string `json:"body_check_expr"`
	Method        string `json:"method"`
}
