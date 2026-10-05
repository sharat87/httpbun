package mix

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func seqNumbers(t *testing.T, args ...int) []int {
	seq, err := tplFuncSeq(args...)
	assert.NoError(t, err)
	ns := []int{}
	for _, item := range seq {
		ns = append(ns, item.N)
	}
	return ns
}

func TestSeq(t *testing.T) {
	s := assert.New(t)
	s.Equal([]int{0, 1, 2, 3, 4}, seqNumbers(t, 5))
	s.Equal([]int{3, 4, 5, 6}, seqNumbers(t, 3, 7))
	s.Equal([]int{2, 5, 8, 11}, seqNumbers(t, 2, 13, 3))
	s.Equal([]int{7, 6, 5, 4}, seqNumbers(t, 7, 3))
	s.Equal([]int{}, seqNumbers(t, 0))
}

func TestSeqFirstAndLast(t *testing.T) {
	s := assert.New(t)
	seq, err := tplFuncSeq(3)
	s.NoError(err)
	s.Equal([]SeqItem{{N: 0, IsFirst: true}, {N: 1}, {N: 2, IsLast: true}}, seq)
}

func TestSeqInvalid(t *testing.T) {
	for _, args := range [][]int{{}, {1, 2, 3, 4}, {0, 5, 0}, {maxSeqLength + 1}} {
		_, err := tplFuncSeq(args...)
		assert.Error(t, err, "args: %v", args)
	}
}
