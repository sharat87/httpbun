package llm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitIntoChunks(t *testing.T) {
	for _, text := range []string{"", "one", "one two", "  lead and trail  ", "line1\n\nline2  x", "tab\there"} {
		chunks := splitIntoChunks(text)
		assert.Equal(t, text, strings.Join(chunks, ""), "chunks: %q", chunks)
	}
	assert.Equal(t, []string{"Hello\n\n", "World"}, splitIntoChunks("Hello\n\nWorld"))
}
