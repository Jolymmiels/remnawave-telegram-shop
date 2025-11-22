package stats

type StatsTotals struct {
	Day   float64 `json:"day"`
	Week  float64 `json:"week"`
	Month float64 `json:"month"`
}

type MonthlyGrowth struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}
type UserGrowthStats struct {
	NewUsersLastMonth int64 `json:"new_users_last_month"`
	TotalUsers        int64 `json:"total_users"`
}
