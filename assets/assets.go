package assets

import (
	"embed"
	"github.com/sharat87/httpbun/mux"
	"log"
	"net/http"
	"strings"
)

//go:embed *.html *.png favicon.ico site.webmanifest
var assets embed.FS

func WriteAsset(name string, w http.ResponseWriter, req *mux.Request) {
	if content, err := assets.ReadFile(name); err == nil {
		w.Write(content)
	} else if strings.HasSuffix(err.Error(), " file does not exist") {
		http.NotFound(w, &req.Request)
	} else {
		log.Printf("Error %v", err)
	}
}
