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
		err := cb(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func ArrEachIdx[T any](vals []T, cb func([]T) []T) {
	cb(vals)
}

func ArrUnique[T comparable](vals []T) []T {
	valsUnique := []T{}
	valsMap := map[T]bool{}
	for _, val := range vals {
		if valsMap[val] {
			continue
		}

		valsUnique = append(valsUnique, val)
		valsMap[val] = true
	}

	return valsUnique
}

func ArrPrepend[T comparable](vals []T, v T) []T {
	if len(vals) == 0 {
		return vals
	}

	var noop T
	vals = append(vals, noop)
	copy(vals[1:], vals)
	vals[0] = v
	return vals
}
