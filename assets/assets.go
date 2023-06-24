package assets

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed *.html *.css *.png *.svg favicon.ico site.webmanifest
var assets embed.FS

func Render(name string, w http.ResponseWriter, data any) {
	tpl, err := template.ParseFS(assets, "*")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}

	var rendered bytes.Buffer
	if err := tpl.ExecuteTemplate(&rendered, name, data); err != nil {
		log.Fatalf("Error executing %q template %v.", name, err)
	}

	_, err = w.Write(rendered.Bytes())
	if err != nil {
		log.Printf("Error writing rendered template %v", err)
	}
}

func WriteAsset(name string, w http.ResponseWriter, req *http.Request) {
	if content, err := assets.ReadFile(name); err == nil {
		_, err := w.Write(content)
		if err != nil {
			log.Printf("Error writing asset content %v", err)
		}
	} else if strings.HasSuffix(err.Error(), " file does not exist") {
		http.NotFound(w, req)
	} else {
		log.Printf("Error %v", err)
	}
}
