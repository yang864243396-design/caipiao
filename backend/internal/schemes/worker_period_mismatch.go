package schemes

import "strings"

func isAcceptedPeriodMismatch(targetPeriod, acceptedPeriod string) bool {
	targetPeriod = strings.TrimSpace(targetPeriod)
	acceptedPeriod = strings.TrimSpace(acceptedPeriod)
	return targetPeriod != "" && acceptedPeriod != "" && targetPeriod != acceptedPeriod
}
