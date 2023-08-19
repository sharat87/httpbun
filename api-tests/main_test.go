package api_tests

import (
	"github.com/sharat87/httpbun/server"
	"github.com/sharat87/httpbun/server/spec"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

//goland:noinspection HttpUrlsUsage
const (
	BindTarget = "127.0.0.1:30001"
	BaseURL    = "http://" + BindTarget + "/"
)

type R struct {
	Method  string
	Path    string
	Body    string
	Headers map[string][]string
}

func ExecRequest(r R) (http.Response, string) {
	var bodyReader io.Reader
	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	if r.Method == "" {
		r.Method = http.MethodGet
	}

	req, err := http.NewRequest(r.Method, BaseURL+r.Path, bodyReader)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "")
	for name, values := range r.Headers {
		req.Header[name] = values
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return *resp, string(bodyText)
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)

	s := server.StartNew(spec.Spec{BindTarget: BindTarget})
	defer s.CloseAndWait()

	m.Run()
}
