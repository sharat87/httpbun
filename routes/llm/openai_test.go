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

	// Long text is grouped into several words per chunk, so it still streams in at most maxStreamChunks chunks.
	for _, words := range []int{maxStreamChunks, maxStreamChunks + 1, 3 * maxStreamChunks, 100_000} {
		text := strings.Repeat("word ", words)
		chunks := splitIntoChunks(text)
		assert.LessOrEqual(t, len(chunks), maxStreamChunks, "words: %d", words)
		assert.Equal(t, text, strings.Join(chunks, ""), "words: %d", words)
	}
	assert.Len(t, splitIntoChunks(strings.Repeat("w ", maxStreamChunks)), maxStreamChunks)
}
