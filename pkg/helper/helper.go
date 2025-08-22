package helper

// convertInt64ToIntSlice mengubah []int64 menjadi []int
func ConvertInt64ToIntSlice(input []int64) []int {
	result := make([]int, len(input))
	for i, v := range input {
		result[i] = int(v)
	}
	return result
}
