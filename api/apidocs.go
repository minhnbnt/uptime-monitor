package apidocs

import "embed"

//go:embed *.yaml paths/*.yaml schemas/*.yaml
var FS embed.FS

//go:embed docs/index.html
var DocsHTML string
