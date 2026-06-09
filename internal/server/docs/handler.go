package docs

import (
	"html/template"
	"net/http"
	"strings"
)

var swaggerUI = template.Must(template.New("swagger").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}} - API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    SwaggerUIBundle({
      url: {{.SpecURL}},
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset,
      ],
      plugins: [
        SwaggerUIBundle.plugins.DownloadUrl,
      ],
      layout: "StandaloneLayout",
    });
  </script>
</body>
</html>`))

func Handler(title string) http.Handler {
	apiFS := http.FileServer(http.Dir("./api"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/docs/" || r.URL.Path == "/docs":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			swaggerUI.Execute(w, map[string]string{
				"Title":   title,
				"SpecURL": "./api/spec.yaml",
			})

		case strings.HasPrefix(r.URL.Path, "/docs/api/"):
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/docs/api/")
			apiFS.ServeHTTP(w, r)

		default:
			http.NotFound(w, r)
		}
	})
}
