package schemes

import "context"

// MaintenanceResumeScheduler 维护解除后主动扫描 pending+maintenance 续投。
type MaintenanceResumeScheduler interface {
	TickMaintenanceResume(ctx context.Context)
}
