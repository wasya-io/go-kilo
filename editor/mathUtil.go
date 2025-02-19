package editor

// mathUtil.go は数学関連のユーティリティ関数を提供します

// max は2つの整数のうち大きい方を返す
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min は2つの整数のうち小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
