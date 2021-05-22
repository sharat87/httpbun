package assets

import (
	"github.com/sharat87/httpbun/request"
	"bytes"
	"log"
	"html/template"
	"embed"
	"strings"
	"net/http"
)

//go:embed *.html *.png favicon.ico site.webmanifest
var assets embed.FS

func Render(w http.ResponseWriter, name string, data interface{}) {
	tpl, err := template.ParseFS(assets, "*.html")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}

	var rendered bytes.Buffer
	if err := tpl.ExecuteTemplate(&rendered, name, data); err != nil {
		log.Fatalf("Error executing %q template %v.", name, err)
	}

	w.Write(rendered.Bytes())
}

func WriteAsset(name string, w http.ResponseWriter, req *request.Request) {
	if content, err := assets.ReadFile("assets/" + name); err == nil {
		w.Write(content)
	} else if strings.HasSuffix(err.Error(), " file does not exist") {
		http.NotFound(w, &req.Request)
	} else {
		log.Printf("Error %v", err)
	}
}
