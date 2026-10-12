// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

// maxDistance is the largest edit distance between a mistyped name and the name that a message
// suggests for it.
const maxDistance = 2

// suggest returns prefix and the candidate closest to name, such as --addr, or the empty string
// when no candidate is within maxDistance edits of name. Of two candidates at one distance, suggest
// returns the first in candidates.
func suggest(name string, candidates []string, prefix string) string {
	best, bestDistance := "", maxDistance+1
	for _, c := range candidates {
		if d := distance(name, c); d < bestDistance {
			best, bestDistance = c, d
		}
	}
	if best == "" {
		return ""
	}
	return prefix + best
}

// distance returns the Levenshtein distance between the bytes of a and b: the least number of
// insertions, deletions and substitutions that turn a into b. It allocates two rows of len(b)+1
// counts.
func distance(a, b string) int {
	prev, cur := make([]int, len(b)+1), make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := range len(a) {
		cur[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			cur[j+1] = min(prev[j+1]+1, cur[j]+1, prev[j]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
