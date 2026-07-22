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

// OpEvent is a single provisioning or update operation used for trend computation.
type OpEvent struct {
	InstanceID string
	CreatedAt  string // YYYY-MM-DD
	Type       string // "provision" or "update"
	RawParams  string // provisioning_parameters JSON for provision ops; updating_parameters JSON for update ops
}

// FetchOpEventsInRange returns all succeeded provisioning and update operations on active
// instances within tr, ordered by created_at ASC. Used for trend (AC6) computation.
func (r *DBReader) FetchOpEventsInRange(tr TimeRange) ([]OpEvent, error) {
	q := `
SELECT o.instance_id, TO_CHAR(o.created_at, 'YYYY-MM-DD') AS created_date, o.type,
       COALESCE(CASE WHEN o.type = 'provision' THEN o.provisioning_parameters::text ELSE o.data->>'updating_parameters' END, '{}') AS raw_params
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

// OpEventsToProvParamsInRange derives ProvisioningParamsWithID from provision events in events,
// filtering to those whose CreatedAt falls within tr. An empty tr returns all events.
func OpEventsToProvParamsInRange(events []OpEvent, tr TimeRange) []ProvisioningParamsWithID {
	result := make([]ProvisioningParamsWithID, 0, len(events))
	for _, ev := range events {
		if ev.Type != "provision" {
			continue
		}
		if !inRange(ev.CreatedAt, tr) {
			continue
		}
		p, err := parseProvisioningParameters(ev.RawParams)
		if err != nil {
			slog.Warn("analytics: skipping malformed provisioning_parameters in op event", "instance_id", ev.InstanceID, "error", err)
			continue
		}
		result = append(result, ProvisioningParamsWithID{InstanceID: ev.InstanceID, Params: p})
	}
	return result
}

// OpEventsToUpdateParamsInRange derives UpdateParamsWithID from update events in events,
// filtering to those whose CreatedAt falls within tr. An empty tr returns all events.
func OpEventsToUpdateParamsInRange(events []OpEvent, tr TimeRange) []UpdateParamsWithID {
	result := make([]UpdateParamsWithID, 0, len(events))
	for _, ev := range events {
		if ev.Type != "update" {
			continue
		}
		if !inRange(ev.CreatedAt, tr) {
			continue
		}
		var params internal.UpdatingParametersDTO
		if err := json.Unmarshal([]byte(ev.RawParams), &params); err != nil {
			slog.Warn("analytics: skipping malformed updating_parameters in op event", "instance_id", ev.InstanceID, "error", err)
			continue
		}
		result = append(result, UpdateParamsWithID{InstanceID: ev.InstanceID, Params: params})
	}
	return result
}

// inRange returns true if the YYYY-MM-DD date string d falls within [tr.From, tr.To).
// An empty tr (both zero) always returns true. Single-bounded ranges are supported:
// a zero From means unbounded start; a zero To means unbounded end.
func inRange(d string, tr TimeRange) bool {
	if tr.From.IsZero() && tr.To.IsZero() {
		return true
	}
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		slog.Warn("analytics: skipping event with unparseable date", "date", d, "error", err)
		return false
	}
	if !tr.From.IsZero() && t.Before(tr.From) {
		return false
	}
	if !tr.To.IsZero() && !t.Before(tr.To) {
		return false
	}
	return true
}
