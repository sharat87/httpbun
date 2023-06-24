package api_tests

import tu "github.com/sharat87/httpbun/test_utils"

func (s *TSuite) TestMissingTargetURL() {
	resp, body := s.ExecRequest(tu.R{
		Method: "GET",
		Path:   "redirect-to",
	})
	s.Equal(400, resp.StatusCode)
	s.Equal("text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	s.Equal("19", resp.Header.Get("Content-Length"))
	s.Equal("Need url parameter\n", string(body))
}
