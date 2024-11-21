package assets

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

//go:embed *
var assets embed.FS

var assetsTemplate = prepare()

func prepare() template.Template {
	t, err := template.ParseFS(assets, "*.html", "*.css", "*.js")
	if err != nil {
		log.Fatalf("Error parsing HTML assets %v.", err)
	}
	return *t
}

func Render(name string, ex exchange.Exchange, data map[string]any) response.Response {
	if data == nil {
		data = make(map[string]any)
	}

	data["pathPrefix"] = ex.ServerSpec.PathPrefix
	data["commit"] = ex.ServerSpec.Commit
	data["date"] = ex.ServerSpec.Date
	data["host"] = ex.Request.URL.Host

	buf := bytes.Buffer{}
	err := assetsTemplate.ExecuteTemplate(&buf, name, data)

	if err != nil {
		log.Fatalf("Error executing %q template %v.", name, err)
	}

	return response.New(
		http.StatusOK,
		http.Header{
			c.ContentType: []string{c.TextHTML},
		},
		buf.Bytes(),
	)
}

func WriteAsset(name string) response.Response {
	file, err := assets.Open(name)
	if err != nil {
		if strings.HasSuffix(err.Error(), " file does not exist") {
			return response.New(http.StatusNotFound, nil, nil)
		} else {
			return response.New(http.StatusInternalServerError, nil, []byte(fmt.Sprintf("Error opening asset file %v", err)))
		}
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing asset file %v", err)
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return response.New(http.StatusInternalServerError, nil, []byte(fmt.Sprintf("Error reading asset file %v", err)))
	}

	return response.New(http.StatusOK, nil, data)
}
