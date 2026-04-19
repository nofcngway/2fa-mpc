package bootstrap

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const scalarHTML = `<!DOCTYPE html>
<html>
<head>
    <title>MPC-2FA API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
        Scalar.createApiReference('#app', {
            sources: [
                { title: 'Auth Service', url: '/openapi/auth.json' },
                { title: 'TwoFA Service', url: '/openapi/twofa.json' }
            ]
        })
    </script>
</body>
</html>`

func DocsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(scalarHTML))
	}
}

func SwaggerFileHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := findSwaggerFile(dir)
		if path == "" {
			http.Error(w, "swagger spec not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, path)
	}
}

func findSwaggerFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if result := findSwaggerFile(full); result != "" {
				return result
			}
		} else if strings.HasSuffix(e.Name(), ".swagger.json") {
			return full
		}
	}
	return ""
}
