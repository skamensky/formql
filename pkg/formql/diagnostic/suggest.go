package diagnostic

import "strings"

// ClosestSuggestion returns the nearest candidate using a conservative edit-distance threshold.
func ClosestSuggestion(target string, candidates []string) (string, bool) {
	normalizedTarget := strings.ToLower(strings.TrimSpace(target))
	if normalizedTarget == "" {
		return "", false
	}

	best := ""
	bestDistance := -1
	for _, candidate := range candidates {
		normalizedCandidate := strings.ToLower(strings.TrimSpace(candidate))
		if normalizedCandidate == "" {
			continue
		}

		distance := levenshtein(normalizedTarget, normalizedCandidate)
		if bestDistance == -1 || distance < bestDistance {
			best = candidate
			bestDistance = distance
		}
	}

	if bestDistance == -1 || bestDistance > suggestionThreshold(normalizedTarget) {
		return "", false
	}
	return best, true
}

func suggestionThreshold(target string) int {
	switch {
	case len(target) <= 4:
		return 1
	case len(target) <= 8:
		return 2
	default:
		return 3
	}
}

func levenshtein(left, right string) int {
	if left == right {
		return 0
	}
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}

	column := make([]int, len(right)+1)
	for y := 1; y <= len(right); y++ {
		column[y] = y
	}

	for x := 1; x <= len(left); x++ {
		column[0] = x
		lastDiagonal := x - 1
		for y := 1; y <= len(right); y++ {
			oldDiagonal := column[y]
			cost := 0
			if left[x-1] != right[y-1] {
				cost = 1
			}

			insertion := column[y] + 1
			deletion := column[y-1] + 1
			substitution := lastDiagonal + cost
			column[y] = minInt(insertion, deletion, substitution)
			lastDiagonal = oldDiagonal
		}
	}

	return column[len(right)]
}

func minInt(values ...int) int {
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
}
