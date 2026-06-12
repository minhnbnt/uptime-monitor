package dto

type ImportRow struct {
	Row          int
	Name         string
	URL          string
	Method       string
	Interval     int
	Timeout      int
	ExpectedCode int
}

type ImportResult struct {
	Imported int
	Errors   []ImportRowError
}

type ImportRowError struct {
	Row     int
	Message string
}
