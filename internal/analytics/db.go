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

const (
	sqlCreatedAtGte = " AND o.created_at >= ?"
	sqlCreatedAtLt  = " AND o.created_at < ?"
)

// TimeRange optionally constrains queries to operations created within [From, To).
// Zero values mean unbounded on that side.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// ProvisioningParamsWithID pairs an instance ID with its provisioning parameters.
type ProvisioningParamsWithID struct {
	InstanceID string
	Params     internal.ProvisioningParameters
}

// UpdateParamsWithID pairs an instance ID with its update parameters.
type UpdateParamsWithID struct {
	InstanceID string
	Params     internal.UpdatingParametersDTO
}

func (r *DBReader) fetchProvisioningParams(tr TimeRange) ([]ProvisioningParamsWithID, error) {
	q := `
SELECT o.instance_id, o.provisioning_parameters
FROM operations o
JOIN instances i ON i.instance_id = o.instance_id
WHERE o.type = 'provision'
  AND o.state = 'succeeded'
  AND i.deleted_at = '0001-01-01 00:00:00+00'`
	args := []interface{}{}
	if !tr.From.IsZero() {
		q += sqlCreatedAtGte
		args = append(args, tr.From)
	}
	if !tr.To.IsZero() {
		q += sqlCreatedAtLt
		args = append(args, tr.To)
	}

	var rows []struct {
		InstanceID             string `db:"instance_id"`
		ProvisioningParameters string `db:"provisioning_parameters"`
	}
	_, err := r.session.SelectBySql(q, args...).Load(&rows)
	if err != nil {
		return nil, fmt.Errorf("fetching active provisioning params: %w", err)
	}

	result := make([]ProvisioningParamsWithID, 0, len(rows))
	for _, row := range rows {
		p, err := parseProvisioningParameters(row.ProvisioningParameters)
		if err != nil {
			slog.Warn("analytics: skipping malformed provisioning_parameters row", "error", err)
			continue
		}
		result = append(result, ProvisioningParamsWithID{InstanceID: row.InstanceID, Params: p})
	}
	return result, nil
}

// FetchActiveProvisioningParams returns ProvisioningParameters for all active instances.
// Active = row exists in instances table with deleted_at = zero (not permanently deprovisioned,
// not failed-deprovision). Temporary deprovisioned instances are considered active.
func (r *DBReader) FetchActiveProvisioningParams() ([]ProvisioningParamsWithID, error) {
	return r.fetchProvisioningParams(TimeRange{})
}

// FetchActiveProvisioningParamsInRange is like FetchActiveProvisioningParams but scoped to tr.
func (r *DBReader) FetchActiveProvisioningParamsInRange(tr TimeRange) ([]ProvisioningParamsWithID, error) {
	return r.fetchProvisioningParams(tr)
}

func (r *DBReader) fetchUpdateParams(tr TimeRange) ([]UpdateParamsWithID, error) {
	q := `
SELECT o.instance_id, o.data
FROM operations o
JOIN instances i ON i.instance_id = o.instance_id
WHERE o.type = 'update'
  AND o.state = 'succeeded'
  AND i.deleted_at = '0001-01-01 00:00:00+00'`
	args := []interface{}{}
	if !tr.From.IsZero() {
		q += sqlCreatedAtGte
		args = append(args, tr.From)
	}
	if !tr.To.IsZero() {
		q += sqlCreatedAtLt
		args = append(args, tr.To)
	}

	var rows []struct {
		InstanceID string `db:"instance_id"`
		Data       string `db:"data"`
	}
	_, err := r.session.SelectBySql(q, args...).Load(&rows)
	if err != nil {
		return nil, fmt.Errorf("fetching update params: %w", err)
	}

	result := make([]UpdateParamsWithID, 0, len(rows))
	for _, row := range rows {
		var op internal.Operation
		if err := json.Unmarshal([]byte(row.Data), &op); err != nil {
			slog.Warn("analytics: skipping malformed operation data row", "error", err)
			continue
		}
		result = append(result, UpdateParamsWithID{InstanceID: row.InstanceID, Params: op.UpdatingParameters})
	}
	return result, nil
}

// FetchUpdateParams returns UpdatingParametersDTO for all update operations on active instances.
func (r *DBReader) FetchUpdateParams() ([]UpdateParamsWithID, error) {
	return r.fetchUpdateParams(TimeRange{})
}

// FetchUpdateParamsInRange is like FetchUpdateParams but scoped to tr.
func (r *DBReader) FetchUpdateParamsInRange(tr TimeRange) ([]UpdateParamsWithID, error) {
	return r.fetchUpdateParams(tr)
}

// OpEvent is a single provisioning or update operation used for trend computation.
type OpEvent struct {
	InstanceID string
	CreatedAt  string // YYYY-MM-DD
	Type       string // "provision" or "update"
	RawParams  string // provisioning_parameters for provision ops; operation data JSON for update ops
}

// FetchOpEventsInRange returns all succeeded provisioning and update operations on active
// instances within tr, ordered by created_at ASC. Used for trend (AC6) computation.
func (r *DBReader) FetchOpEventsInRange(tr TimeRange) ([]OpEvent, error) {
	q := `
SELECT o.instance_id, DATE(o.created_at) AS created_date, o.type,
       CASE WHEN o.type = 'provision' THEN o.provisioning_parameters ELSE o.data END AS raw_params
FROM operations o
JOIN instances i ON i.instance_id = o.instance_id
WHERE o.type IN ('provision', 'update')
  AND o.state = 'succeeded'
  AND i.deleted_at = '0001-01-01 00:00:00+00'`
	args := []interface{}{}
	if !tr.From.IsZero() {
		q += sqlCreatedAtGte
		args = append(args, tr.From)
	}
	if !tr.To.IsZero() {
		q += sqlCreatedAtLt
		args = append(args, tr.To)
	}
	q += " ORDER BY o.created_at ASC"

	var rows []struct {
		InstanceID  string `db:"instance_id"`
		CreatedDate string `db:"created_date"`
		Type        string `db:"type"`
		RawParams   string `db:"raw_params"`
	}
	_, err := r.session.SelectBySql(q, args...).Load(&rows)
	if err != nil {
		return nil, fmt.Errorf("fetching op events: %w", err)
	}

	result := make([]OpEvent, len(rows))
	for i, row := range rows {
		result[i] = OpEvent{
			InstanceID: row.InstanceID,
			CreatedAt:  row.CreatedDate,
			Type:       row.Type,
			RawParams:  row.RawParams,
		}
	}
	return result, nil
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

// PlainProvisioningParams extracts just the ProvisioningParameters slice from ProvisioningParamsWithID.
func PlainProvisioningParams(params []ProvisioningParamsWithID) []internal.ProvisioningParameters {
	result := make([]internal.ProvisioningParameters, len(params))
	for i, p := range params {
		result[i] = p.Params
	}
	return result
}

// PlainUpdateParams extracts just the UpdatingParametersDTO slice from UpdateParamsWithID.
func PlainUpdateParams(params []UpdateParamsWithID) []internal.UpdatingParametersDTO {
	result := make([]internal.UpdatingParametersDTO, len(params))
	for i, p := range params {
		result[i] = p.Params
	}
	return result
}
