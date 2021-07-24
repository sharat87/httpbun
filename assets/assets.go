package assets

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed *.html *.css *.png favicon.ico site.webmanifest
var assets embed.FS

func Render(name string, w http.ResponseWriter, data interface{}) {
	tpl, err := template.ParseFS(assets, "*")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}

	var rendered bytes.Buffer
	if err := tpl.ExecuteTemplate(&rendered, name, data); err != nil {
		log.Fatalf("Error executing %q template %v.", name, err)
	}

	w.Write(rendered.Bytes())
}

func WriteAsset(name string, w http.ResponseWriter, req *http.Request) {
	if content, err := assets.ReadFile(name); err == nil {
		w.Write(content)
	} else if strings.HasSuffix(err.Error(), " file does not exist") {
		http.NotFound(w, req)
	} else {
		log.Printf("Error %v", err)
	}
}
