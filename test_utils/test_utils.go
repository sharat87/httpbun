package test_utils

//goland:noinspection HttpUrlsUsage
const (
	BindTarget = "127.0.0.1:30001"
	baseURL    = "http://" + BindTarget + "/"
)

type R struct {
	Method  string
	Path    string
	Body    string
	Headers map[string][]string
}
