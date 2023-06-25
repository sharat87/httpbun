package test_utils

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func ExecRequest(r R) (http.Response, string) {
	var bodyReader io.Reader
	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	req, err := http.NewRequest(r.Method, baseURL+r.Path, bodyReader)
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

	return *resp, string(bodyText)
}
