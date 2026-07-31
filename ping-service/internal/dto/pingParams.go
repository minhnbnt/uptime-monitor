package dto

type CheckParams struct {
	K8sObjectCheckParams
	HTTPCheckParams *HTTPCheckParams
}

type K8sObjectCheckParams struct {
	Namespace     string
	Kind          string
	ObjectID      string
	ContainerName string // "" means no container check
}

type HTTPCheckParams struct {
	Method        string
	Port          int
	EndpointPath  string
	ExpectedCode  int
	BodyCheckExpr string // "" means no body check
}
