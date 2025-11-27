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

type StatsOverview struct {
	Users    UserStats    `json:"users"`
	Revenue  RevenueStats `json:"revenue"`
	Payments PaymentStats `json:"payments"`
}

type UserStats struct {
	Total       int64 `json:"total"`
	Active      int64 `json:"active"`
	Expired     int64 `json:"expired"`
	Blocked     int64 `json:"blocked"`
	NewToday    int64 `json:"new_today"`
	NewThisWeek int64 `json:"new_this_week"`
	NewThisMonth int64 `json:"new_this_month"`
}

type RevenueStats struct {
	Today      float64 `json:"today"`
	ThisWeek   float64 `json:"this_week"`
	ThisMonth  float64 `json:"this_month"`
	AllTime    float64 `json:"all_time"`
	AvgCheck   float64 `json:"avg_check"`
}

type PaymentStats struct {
	TotalCount      int64            `json:"total_count"`
	TodayCount      int64            `json:"today_count"`
	ByCurrency      []CurrencyStat   `json:"by_currency"`
	ByPaymentType   []PaymentTypeStat `json:"by_payment_type"`
}

type CurrencyStat struct {
	Currency string  `json:"currency"`
	Count    int64   `json:"count"`
	Amount   float64 `json:"amount"`
}

type PaymentTypeStat struct {
	Type   string  `json:"type"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

type DailyGrowth struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type DailyRevenue struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}
