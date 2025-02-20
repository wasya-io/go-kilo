package editor

// Max は2つの整数の大きい方を返す
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min は2つの整数の小さい方を返す
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Abs は整数の絶対値を返す
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
