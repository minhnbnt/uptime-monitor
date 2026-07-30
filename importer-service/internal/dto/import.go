package dto

type ImportRow struct {
	Row           int
	Name          string
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string
	Interval      int
	Timeout       int
}

type ImportSuccess struct {
	Row      int
	Name     string
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
