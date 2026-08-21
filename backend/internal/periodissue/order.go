// Package periodissue compares provider issue identifiers without assuming
// they fit in a fixed-width integer.
package periodissue

import "strings"

// Advances reports whether current is a strictly newer issue than previous.
// Comparable issues have a non-empty decimal suffix and an identical prefix.
// Incomparable or numerically equal identifiers are deliberately rejected.
func Advances(previous, current string) bool {
	previous = strings.TrimSpace(previous)
	current = strings.TrimSpace(current)
	previousPrefix, previousSequence, previousOK := trailingDecimal(previous)
	currentPrefix, currentSequence, currentOK := trailingDecimal(current)
	if !previousOK || !currentOK || previousPrefix != currentPrefix {
		return false
	}
	previousSequence = trimDecimalZeros(previousSequence)
	currentSequence = trimDecimalZeros(currentSequence)
	if len(currentSequence) != len(previousSequence) {
		return len(currentSequence) > len(previousSequence)
	}
	return currentSequence > previousSequence
}

func trailingDecimal(issue string) (prefix, sequence string, ok bool) {
	sequenceStart := len(issue)
	for sequenceStart > 0 {
		char := issue[sequenceStart-1]
		if char < '0' || char > '9' {
			break
		}
		sequenceStart--
	}
	if sequenceStart == len(issue) {
		return "", "", false
	}
	return issue[:sequenceStart], issue[sequenceStart:], true
}

func trimDecimalZeros(sequence string) string {
	for len(sequence) > 1 && sequence[0] == '0' {
		sequence = sequence[1:]
	}
	return sequence
}
