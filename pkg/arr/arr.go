package arr

func ArrMap[T any, K any](vals []T, cb func(T) K) []K {
	result := make([]K, len(vals))
	for i, v := range vals {
		result[i] = cb(v)
	}

	return result
}

func ArrEach[T any](vals []T, cb func(T)) {
	for _, v := range vals {
		cb(v)
	}
}

func ArrEachWithErr[T any](vals []T, cb func(T) error) error {
	for _, v := range vals {
		if err := cb(v); err != nil {
			return err
		}
	}

	return nil
}

// ArrUnique returns vals without duplicates, keeping the first occurrence order.
func ArrUnique[T comparable](vals []T) []T {
	valsUnique := make([]T, 0, len(vals))
	seen := make(map[T]struct{}, len(vals))
	for _, val := range vals {
		if _, ok := seen[val]; ok {
			continue
		}

		valsUnique = append(valsUnique, val)
		seen[val] = struct{}{}
	}

	return valsUnique
}

// ArrPrepend returns a new slice with v followed by vals. The input slice is
// never modified.
func ArrPrepend[T any](vals []T, v T) []T {
	result := make([]T, 0, len(vals)+1)
	result = append(result, v)
	return append(result, vals...)
}
