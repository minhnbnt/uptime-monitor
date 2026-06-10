package apidocs

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed docs/index.html
var docsHTML string

//go:embed *.yaml paths/*.yaml schemas/*.yaml
var specFS embed.FS

func GetHandler(title string) (http.Handler, error) {

	swaggerUI, err := template.New("swagger").Parse(docsHTML)
	if err != nil {
		return nil, err
	}

	sub, err := fs.Sub(specFS, ".")
	if err != nil {
		return nil, err
	}

	specsServer := http.FileServer(http.FS(sub))

	mux := http.NewServeMux()

	mux.Handle("/api/", http.StripPrefix("/api/", specsServer))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := swaggerUI.Execute(w, map[string]string{
			"Title":   title,
			"SpecURL": "./api/spec.yaml",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return mux, nil
}
