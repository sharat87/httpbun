package static

import (
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
	"github.com/sharat87/httpbun/response"
	"net/http"
)

var Routes = map[string]exchange.HandlerFn{
	"/deny":       handleRobotsDeny,
	"/robots.txt": handleRobotsTxt,
	"/html":       handleHtml,
}

func handleRobotsTxt(ex *exchange.Exchange) response.Response {
	return response.New(http.StatusOK, http.Header{
		c.ContentType: []string{c.TextPlain},
	}, []byte("User-agent: *\nDisallow: /deny\nDisallow: /mix/\nDisallow: /run/"))
}

func handleRobotsDeny(ex *exchange.Exchange) response.Response {
	return response.New(http.StatusOK, http.Header{
		c.ContentType: []string{c.TextPlain},
	}, []byte(`
          .-''''''-.
        .' _      _ '.
       /   O      O   \
      :                :
      |                |
      :       __       :
       \  .-"`+"`  `"+`"-.  /
        '.          .'
          '-......-'
     YOU SHOULDN'T BE HERE`))
}

func handleHtml(ex *exchange.Exchange) response.Response {
	return response.New(http.StatusOK, http.Header{
		c.ContentType: []string{c.TextHTML},
	}, []byte(`<!DOCTYPE html>
<html>
<title>Httpbun sample</title>
<body>
  <h1>Some title</h1>
  <p>Some paragraph</p>
  <img src=x onerror='document.body.insertAdjacentText("beforeend", "inserted by img[onerror]")'>
  <script>document.write("inserted by script")</script>
`))
}
