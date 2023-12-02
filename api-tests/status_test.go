package api_tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStatus(t *testing.T) {
	s := assert.New(t)
	for i := 200; i <= 599; i++ {
		resp, _ := ExecRequest(R{
			Path: fmt.Sprintf("status/%d", i),
		})
		s.Equal(i, resp.StatusCode)
	}
}
