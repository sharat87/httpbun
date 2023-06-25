package api_tests

import (
	"encoding/json"
	"github.com/sharat87/httpbun/server"
	tu "github.com/sharat87/httpbun/test_utils"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TSuite))
}

type TSuite struct {
	suite.Suite
	server server.Server
}

func (s *TSuite) SetupSuite() {
	s.server = server.StartNew(server.Config{
		BindTarget: "127.0.0.1:30001",
	})
}

func (s *TSuite) TearDownSuite() {
	s.server.CloseAndWait()
}

func (s *TSuite) ExecRequest(r tu.R) (http.Response, []byte) {
	var bodyReader io.Reader
	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	//goland:noinspection HttpUrlsUsage
	req, err := http.NewRequest(r.Method, "http://"+s.server.Addr.String()+"/"+r.Path, bodyReader)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "httpbun-tests")
	for name, values := range r.Headers {
		req.Header[name] = values
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
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

	return *resp, bodyText
}

func parseJSON(body []byte) map[string]any {
	parsedBody := map[string]any{}
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		log.Fatal(err)
	}
	return parsedBody
}
