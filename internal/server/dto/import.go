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

type ImportRowError struct {
	Row     int
	Message string
}

type ImportError struct {
	Message string
}

type ImportResult struct {
	Imported    int
	RowErrors   []ImportRowError
	BatchErrors []ImportError
}
