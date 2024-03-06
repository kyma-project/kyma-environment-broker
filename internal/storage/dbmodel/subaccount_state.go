package dbmodel

type SubaccountStateDTO struct {
	ID string `json:"id"`

	BetaEnabled       string `json:"beta_enabled"`
	UsedForProduction string `json:"used_for_production"`

	ModifiedAt int64 `json:"modified_at"`
}
