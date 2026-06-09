package docs

import (
	"html/template"
	"io/fs"
	"net/http"

	apidocs "github.com/minhnbnt/uptime-monitor/api"
)


func Handler(title string) http.Handler {

	swaggerUI := template.Must(template.New("swagger").Parse(apidocs.DocsHTML))

	sub, err := fs.Sub(apidocs.FS, ".")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/api/", http.StripPrefix("/api/", http.FileServer(http.FS(sub))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		swaggerUI.Execute(w, map[string]string{
			"Title":   title,
			"SpecURL": "./api/spec.yaml",
		})
	})

	return mux
}
