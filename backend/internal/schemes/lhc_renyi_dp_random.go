package schemes

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// normalizeLHCRenyiDuipengRandomCounts keeps the random-draw configuration in
// the same two-zone shape as its bet content. Legacy counts=[total] splits the
// total evenly with the extra pick assigned to B; explicit A/B values retain A
// first and clamp B to the remaining shared cap.
func normalizeLHCRenyiDuipengRandomCounts(counts []int) []int {
	if len(counts) == 1 {
		total := counts[0]
		if total < 2 {
			total = 2
		}
		if total > lhcRenyiDuipengMaxPicks {
			total = lhcRenyiDuipengMaxPicks
		}
		a := total / 2
		return []int{a, total - a}
	}

	a, b := 1, 1
	if len(counts) > 0 && counts[0] > 0 {
		a = counts[0]
	}
	if len(counts) > 1 && counts[1] > 0 {
		b = counts[1]
	}
	if a > lhcRenyiDuipengMaxPicks-1 {
		a = lhcRenyiDuipengMaxPicks - 1
	}
	if b > lhcRenyiDuipengMaxPicks-a {
		b = lhcRenyiDuipengMaxPicks - a
	}
	return []int{a, b}
}

// randomLHCRenyiDuipengContent draws both zones from one shuffled 01-49 pool,
// so a number can never appear in both A and B.
func randomLHCRenyiDuipengContent(aCount, bCount int) string {
	counts := normalizeLHCRenyiDuipengRandomCounts([]int{aCount, bCount})
	aCount, bCount = counts[0], counts[1]
	perm := rand.Perm(49)
	a := append([]int(nil), perm[:aCount]...)
	b := append([]int(nil), perm[aCount:aCount+bCount]...)
	sort.Ints(a)
	sort.Ints(b)
	format := func(nums []int) string {
		out := make([]string, 0, len(nums))
		for _, n := range nums {
			out = append(out, fmt.Sprintf("%02d", n+1))
		}
		return strings.Join(out, ",")
	}
	return format(a) + "|" + format(b)
}
