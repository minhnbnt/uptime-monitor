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

type ImportSuccess struct {
	Row      int
	Name     string
	URL      string
	ServerID uint
}

type ImportRowError struct {
	Row     int
	Message string
}

type ImportError struct {
	Message string
}

type ImportResult struct {
	Successes   []ImportSuccess
	RowErrors   []ImportRowError
	BatchErrors []ImportError
}
