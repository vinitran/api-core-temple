package arr

import (
	"slices"
	"testing"
)

func TestArrPrepend(t *testing.T) {
	t.Parallel()

	if got := ArrPrepend([]int{}, 1); !slices.Equal(got, []int{1}) {
		t.Errorf("prepend to empty = %v, want [1]", got)
	}

	in := make([]int, 2, 10)
	in[0], in[1] = 2, 3
	got := ArrPrepend(in, 1)
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("prepend = %v, want [1 2 3]", got)
	}
	if !slices.Equal(in, []int{2, 3}) {
		t.Errorf("input mutated: %v", in)
	}
}

func TestArrUnique(t *testing.T) {
	t.Parallel()

	if got := ArrUnique([]string{"a", "b", "a", "c", "b"}); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("unique = %v", got)
	}
}
