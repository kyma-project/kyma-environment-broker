package analytics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

// DBReader wraps a raw dbr session for analytics queries.
type DBReader struct {
	session *dbr.Session
}

// NewDBReader creates a DBReader from a dbr session.
func NewDBReader(session *dbr.Session) *DBReader {
	return &DBReader{session: session}
}

// TimeRange optionally constrains queries to operations created within [From, To).
// Zero values mean unbounded on that side.
type TimeRange struct {
	From time.Time
	To   time.Time
}

func (r *DBReader) fetchProvisioningParams(tr TimeRange) ([]internal.ProvisioningParameters, error) {
	q := `
SELECT o.provisioning_parameters
FROM operations o
JOIN instances i ON i.instance_id = o.instance_id
WHERE o.type = 'provision'
  AND o.state = 'succeeded'
  AND i.deleted_at = '0001-01-01 00:00:00+00'`
	args := []interface{}{}
	if !tr.From.IsZero() {
		q += " AND o.created_at >= ?"
		args = append(args, tr.From)
	}
	if !tr.To.IsZero() {
		q += " AND o.created_at < ?"
		args = append(args, tr.To)
	}

	var rows []struct {
		ProvisioningParameters string `db:"provisioning_parameters"`
	}
	_, err := r.session.SelectBySql(q, args...).Load(&rows)
	if err != nil {
		return nil, fmt.Errorf("fetching active provisioning params: %w", err)
	}

	result := make([]internal.ProvisioningParameters, 0, len(rows))
	for _, row := range rows {
		p, err := parseProvisioningParameters(row.ProvisioningParameters)
		if err != nil {
			slog.Warn("analytics: skipping malformed provisioning_parameters row", "error", err)
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

// FetchActiveProvisioningParams returns ProvisioningParameters for all active instances.
// Active = row exists in instances table with deleted_at = zero (not permanently deprovisioned,
// not failed-deprovision). Temporary deprovisioned instances are considered active.
func (r *DBReader) FetchActiveProvisioningParams() ([]internal.ProvisioningParameters, error) {
	return r.fetchProvisioningParams(TimeRange{})
}

// FetchActiveProvisioningParamsInRange is like FetchActiveProvisioningParams but scoped to tr.
func (r *DBReader) FetchActiveProvisioningParamsInRange(tr TimeRange) ([]internal.ProvisioningParameters, error) {
	return r.fetchProvisioningParams(tr)
}

func (r *DBReader) fetchUpdateParams(tr TimeRange) ([]internal.UpdatingParametersDTO, error) {
	q := `
SELECT o.data
FROM operations o
JOIN instances i ON i.instance_id = o.instance_id
WHERE o.type = 'update'
  AND o.state = 'succeeded'
  AND i.deleted_at = '0001-01-01 00:00:00+00'`
	args := []interface{}{}
	if !tr.From.IsZero() {
		q += " AND o.created_at >= ?"
		args = append(args, tr.From)
	}
	if !tr.To.IsZero() {
		q += " AND o.created_at < ?"
		args = append(args, tr.To)
	}

	var rows []struct {
		Data string `db:"data"`
	}
	_, err := r.session.SelectBySql(q, args...).Load(&rows)
	if err != nil {
		return nil, fmt.Errorf("fetching update params: %w", err)
	}

	result := make([]internal.UpdatingParametersDTO, 0, len(rows))
	for _, row := range rows {
		var op internal.Operation
		if err := json.Unmarshal([]byte(row.Data), &op); err != nil {
			slog.Warn("analytics: skipping malformed operation data row", "error", err)
			continue
		}
		result = append(result, op.UpdatingParameters)
	}
	return result, nil
}

// FetchUpdateParams returns UpdatingParametersDTO for all update operations on active instances.
func (r *DBReader) FetchUpdateParams() ([]internal.UpdatingParametersDTO, error) {
	return r.fetchUpdateParams(TimeRange{})
}

// FetchUpdateParamsInRange is like FetchUpdateParams but scoped to tr.
func (r *DBReader) FetchUpdateParamsInRange(tr TimeRange) ([]internal.UpdatingParametersDTO, error) {
	return r.fetchUpdateParams(tr)
}

func parseProvisioningParameters(raw string) (internal.ProvisioningParameters, error) {
	if raw == "" {
		return internal.ProvisioningParameters{}, fmt.Errorf("empty provisioning_parameters")
	}
	var p internal.ProvisioningParameters
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return internal.ProvisioningParameters{}, fmt.Errorf("parsing provisioning_parameters: %w", err)
	}
	return p, nil
}
