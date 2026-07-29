package sync

// longestCommonSubsequence finds the LCS of two string slices.
func longestCommonSubsequence(a, b []string) []string {
	lenA, lenB := len(a), len(b)
	if lenA == 0 || lenB == 0 {
		return nil
	}

	dp := make([][]int, lenA+1)
	for i := range dp {
		dp[i] = make([]int, lenB+1)
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	lcs := make([]string, dp[lenA][lenB])
	i, j, idx := lenA, lenB, dp[lenA][lenB]-1
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[idx] = a[i-1]
			i--
			j--
			idx--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}
