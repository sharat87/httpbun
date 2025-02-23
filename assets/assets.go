package assets

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/ex"
	"github.com/sharat87/httpbun/response"
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

func Render(name string, ex ex.Exchange, data map[string]any) response.Response {
	if data == nil {
		data = make(map[string]any)
	}

	data["spec"] = ex.ServerSpec

	data["pathPrefix"] = ex.ServerSpec.PathPrefix

	data["bannerText"] = ex.ServerSpec.Banner
	data["bannerColor"] = ex.ServerSpec.BannerBg

	data["commit"] = ex.ServerSpec.Commit
	data["commitShort"] = ex.ServerSpec.CommitShort
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

func WriteAsset(name string) *response.Response {
	file, err := assets.Open(name)
	if err != nil {
		if strings.HasSuffix(err.Error(), " file does not exist") {
			return &response.Response{Status: http.StatusNotFound}
		} else {
			return &response.Response{
				Status: http.StatusInternalServerError,
				Body:   fmt.Sprintf("Error opening asset file %v", err),
			}
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
		return &response.Response{
			Status: http.StatusInternalServerError,
			Body:   fmt.Sprintf("Error reading asset file %v", err),
		}
	}

	return &response.Response{Body: data}
}
