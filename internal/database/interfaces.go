package database

import (
	"context"
	"time"

	"remnawave-tg-shop-bot/internal/stats"
)

// CustomerRepo defines the interface for customer repository operations.
// Implementations must be safe for concurrent use.
type CustomerRepo interface {
	FindById(ctx context.Context, id int64) (*Customer, error)
	FindByTelegramId(ctx context.Context, telegramId int64) (*Customer, error)
	FindByTelegramIds(ctx context.Context, telegramIds []int64) ([]Customer, error)
	FindByExpirationRange(ctx context.Context, startDate, endDate time.Time) (*[]Customer, error)
	FindAll(ctx context.Context) (*[]Customer, error)
	FindAllWithLanguage(ctx context.Context, language string) (*[]Customer, error)
	FindNonExpired(ctx context.Context) (*[]Customer, error)
	FindNonExpiredWithLanguage(ctx context.Context, language string) (*[]Customer, error)
	FindExpired(ctx context.Context) (*[]Customer, error)
	FindExpiredWithLanguage(ctx context.Context, language string) (*[]Customer, error)
	FindNoSubscription(ctx context.Context) (*[]Customer, error)
	FindNoSubscriptionWithLanguage(ctx context.Context, language string) (*[]Customer, error)
	Create(ctx context.Context, customer *Customer) (*Customer, error)
	FindOrCreate(ctx context.Context, customer *Customer) (*Customer, error)
	CreateBatch(ctx context.Context, customers []Customer) error
	UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error
	UpdateBatch(ctx context.Context, customers []Customer) error
	DeleteByTelegramId(ctx context.Context, telegramID int64) error
	DeleteByNotInTelegramIds(ctx context.Context, telegramIDs []int64) error
	GetUserGrowthStats(ctx context.Context) (*UserGrowthStats, error)
	GetDistinctLanguages(ctx context.Context) ([]string, error)
	GetUserStats(ctx context.Context) (*stats.UserStats, error)
	GetDailyUserGrowth(ctx context.Context, days int) ([]stats.DailyGrowth, error)
}

// PurchaseRepo defines the interface for purchase repository operations.
type PurchaseRepo interface {
	Create(ctx context.Context, purchase *Purchase) (int64, error)
	FindById(ctx context.Context, id int64) (*Purchase, error)
	FindByCustomerID(ctx context.Context, customerID int64) ([]Purchase, error)
	FindByCustomerIDAndInvoiceTypeLast(ctx context.Context, customerID int64, invoiceType InvoiceType) (*Purchase, error)
	FindByInvoiceTypeAndStatus(ctx context.Context, invoiceType InvoiceType, status PurchaseStatus) ([]Purchase, error)
	FindSuccessfulPaidPurchaseByCustomer(ctx context.Context, customerID int64) (*Purchase, error)
	FindLastPaidPurchaseWithPlan(ctx context.Context, customerID int64) (*Purchase, error)
	FindLatestActiveTributesByCustomerIDs(ctx context.Context, customerIDs []int64) ([]Purchase, error)
	UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error
	MarkAsPaid(ctx context.Context, id int64) error
	LockForProcessing(ctx context.Context, id int64) (*Purchase, error)
	UnlockPurchase(ctx context.Context, id int64) error
	CountByPlanID(ctx context.Context, planID int64) (int64, error)
	GetTotalAmountByDateRange(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetMonthlyGrowthLastYear(ctx context.Context) ([]stats.MonthlyGrowth, error)
	GetRevenueStats(ctx context.Context) (*stats.RevenueStats, error)
	GetPaymentStats(ctx context.Context) (*stats.PaymentStats, error)
	GetDailyRevenue(ctx context.Context, days int) ([]stats.DailyRevenue, error)
}

// PlanRepo defines the interface for plan repository operations.
type PlanRepo interface {
	FindAll(ctx context.Context) ([]Plan, error)
	FindActive(ctx context.Context) ([]Plan, error)
	FindById(ctx context.Context, id int64) (*Plan, error)
	FindDefault(ctx context.Context) (*Plan, error)
	FindByName(ctx context.Context, name string) (*Plan, error)
	Create(ctx context.Context, plan *Plan) (*Plan, error)
	Update(ctx context.Context, plan *Plan) (*Plan, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) error
}

// SettingsRepo defines the interface for settings repository operations.
type SettingsRepo interface {
	LoadAll(ctx context.Context) error
	Get(key string) string
	GetInt(key string, defaultValue int) int
	GetBool(key string, defaultValue ...bool) bool
	GetFloat(key string, defaultValue float64) float64
	GetAll() map[string]string
	GetAllFromDB(ctx context.Context) ([]Setting, error)
	Set(ctx context.Context, key, value string) error
	SetMultiple(ctx context.Context, settings map[string]string) error
}

// ReferralRepo defines the interface for referral repository operations.
type ReferralRepo interface {
	Create(ctx context.Context, referrerID, refereeID int64) (*Referral, error)
	FindByReferrer(ctx context.Context, referrerID int64) ([]Referral, error)
	FindByReferee(ctx context.Context, refereeID int64) (*Referral, error)
	CountByReferrer(ctx context.Context, referrerID int64) (int, error)
	MarkBonusGranted(ctx context.Context, referralID int64) error
	CreateBonusHistory(ctx context.Context, referralID int64, purchaseID *int64, bonusDays int, isFirstBonus bool) (*ReferralBonusHistory, error)
	GetBonusHistoryByReferrer(ctx context.Context, referrerID int64) ([]ReferralBonusHistory, error)
}

// BroadcastRepo defines the interface for broadcast repository operations.
type BroadcastRepo interface {
	GetByID(ctx context.Context, id int64) (*Broadcast, error)
	List(ctx context.Context, params BroadcastListParams) (*[]Broadcast, error)
	CreateBroadcast(ctx context.Context, broadcast *Broadcast) (*Broadcast, error)
	Delete(ctx context.Context, id int64) error
	UpdateBroadcastStats(ctx context.Context, id int64, status string, total, sent, failed, blocked int) error
}

// PromoRepo defines the interface for promo repository operations.
type PromoRepo interface {
	Create(ctx context.Context, req *CreatePromoRequest) (*Promo, error)
	GetByID(ctx context.Context, id int64) (*Promo, error)
	GetByCode(ctx context.Context, code string) (*Promo, error)
	GetAll(ctx context.Context) ([]*Promo, error)
	Update(ctx context.Context, id int64, active bool) error
	Delete(ctx context.Context, id int64) error
	HasCustomerUsedPromo(ctx context.Context, promoID, customerID int64) (bool, error)
	RecordPromoUsage(ctx context.Context, promoID, customerID int64) error
}

// Compile-time checks to ensure implementations satisfy interfaces.
var (
	_ CustomerRepo  = (*CustomerRepository)(nil)
	_ PurchaseRepo  = (*PurchaseRepository)(nil)
	_ PlanRepo      = (*PlanRepository)(nil)
	_ SettingsRepo  = (*SettingsRepository)(nil)
	_ ReferralRepo  = (*ReferralRepository)(nil)
	_ BroadcastRepo = (*BroadcastRepository)(nil)
	_ PromoRepo     = (*PromoRepository)(nil)
)
