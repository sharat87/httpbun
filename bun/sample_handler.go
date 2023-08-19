package bun

import (
	"github.com/sharat87/httpbun/assets"
	"github.com/sharat87/httpbun/c"
	"github.com/sharat87/httpbun/exchange"
)

func handleImageSvg(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set(c.ContentType, "image/svg+xml")
	assets.WriteAsset("svg-logo.svg", *ex)
}

func handleSampleRobotsTxt(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set(c.ContentType, c.TextPlain)
	ex.WriteLn("User-agent: *\nDisallow: /deny")
}

func handleSampleRobotsDeny(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set(c.ContentType, c.TextPlain)
	ex.WriteLn(`
          .-''''''-.
        .' _      _ '.
       /   O      O   \
      :                :
      |                |
      :       __       :
       \  .-"` + "`  `" + `"-.  /
        '.          .'
          '-......-'
     YOU SHOULDN'T BE HERE`)
}

func handleSampleHtml(ex *exchange.Exchange) {
	ex.ResponseWriter.Header().Set(c.ContentType, c.TextHTML)
	ex.WriteLn(`<!DOCTYPE html>
<html>
<title>Httpbun sample</title>
<body>
  <h1>Some title</h1>
  <p>Some paragraph</p>
  <img src=x onerror='document.body.insertAdjacentText("beforeend", "inserted by img[onerror]")'>
  <script>document.write("inserted by script")</script>
`)
}
