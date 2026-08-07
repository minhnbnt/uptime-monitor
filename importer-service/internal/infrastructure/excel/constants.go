package excel

var headers = []string{

	"server_name",

	"namespace", "kind",
	"object_id", "container_name",

	"interval_sec",
	"timeout_sec",

	"http_port", "http_path",
	"http_expected_code",
	"http_body_check", "http_method",
}

var expectedHeaders = []string{
	"server_name",
	"namespace", "kind", "object_id",
	"interval_sec",
	"timeout_sec",
}

var headerColumns = func() map[string]int {

	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i + 1
	}

	return m
}()
