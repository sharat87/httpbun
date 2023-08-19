package assets

import (
	"embed"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed *.html *.css *.js *.png *.svg favicon.ico site.webmanifest
var assets embed.FS

var assetsTemplate = prepare()

func prepare() template.Template {
	t, err := template.ParseFS(assets, "*")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}
	return *t
}

func Render(name string, ex exchange.Exchange, data map[string]any) {
	data["serverSpec"] = ex.ServerSpec

	ex.ResponseWriter.Header().Set(c.ContentType, c.TextHTML)

	if err := assetsTemplate.ExecuteTemplate(ex.ResponseWriter, name, data); err != nil {
		log.Fatalf("Error executing %q template %v.", name, err)
	}
}

func WriteAsset(name string, ex exchange.Exchange) {
	file, err := assets.Open(name)
	if err != nil {
		if strings.HasSuffix(err.Error(), " file does not exist") {
			http.NotFound(ex.ResponseWriter, ex.Request)
		} else {
			log.Printf("Error opening asset file %v", err)
		}
		return
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing asset file %v", err)
		}
	}(file)

	_, err = io.Copy(ex.ResponseWriter, file)
	if err != nil {
		log.Printf("Error writing asset content %v", err)
	}
}
