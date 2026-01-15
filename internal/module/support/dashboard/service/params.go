// internal/module/support/dashboard/service/params.go
package dashboard_service

import "github.com/google/uuid"

type GetSCurveParams struct {
	ProjectID uuid.UUID
}

type GetStatsParams struct {
	ProjectID uuid.UUID
}
