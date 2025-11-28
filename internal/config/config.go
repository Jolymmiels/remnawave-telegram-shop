package config

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type config struct {
	telegramToken                                             string
	price1, price3, price6, price12                           int
	starsPrice1, starsPrice3, starsPrice6, starsPrice12       int
	remnawaveUrl, remnawaveToken, remnawaveMode, remnawaveTag string
	defaultLanguage                                           string
	databaseURL                                               string
	cryptoPayURL, cryptoPayToken                              string
	botURL                                                    string
	yookasaURL, yookasaShopId, yookasaSecretKey, yookasaEmail string
	trafficLimit, trialTrafficLimit                           int
	feedbackURL                                               string
	channelURL                                                string
	serverStatusURL                                           string
	supportURL                                                string
	tosURL                                                    string
	isYookasaEnabled                                          bool
	isCryptoEnabled                                           bool
	isTelegramStarsEnabled                                    bool
	adminTelegramId                                           int64
	trialDays                                                 int
	trialRemnawaveTag                                         string
	squadUUIDs                                                map[uuid.UUID]uuid.UUID
	referralDays                                              int
	miniApp                                                   string
	enableAutoPayment                                         bool
	healthCheckPort                                           int
	tributeWebhookUrl, tributeAPIKey, tributePaymentUrl       string
	isWebAppLinkEnabled                                       bool
	daysInMonth                                               int
	externalSquadUUID                                         uuid.UUID
	blockedTelegramIds                                        map[int64]bool
	whitelistedTelegramIds                                    map[int64]bool
	requirePaidPurchaseForStars                               bool
	trialInternalSquads                                       map[uuid.UUID]uuid.UUID
	trialExternalSquadUUID                                    uuid.UUID
	remnawaveHeaders                                          map[string]string
	botAdminURL                                               string
	trialTrafficLimitResetStrategy                            string
	trafficLimitResetStrategy                                 string
}

var conf config

// SettingsProvider interface for database settings
type SettingsProvider interface {
	Get(key string) string
	GetBool(key string) bool
	GetInt(key string, defaultValue int) int
	GetFloat(key string, defaultValue float64) float64
}

var settingsProvider SettingsProvider
var deprecationWarned = make(map[string]bool)

// SetSettingsProvider sets the database settings provider
func SetSettingsProvider(sp SettingsProvider) {
	settingsProvider = sp
}

// getSettingWithDeprecation checks DB first, then falls back to env with deprecation warning
func getSettingWithDeprecation(key string, envValue string, envName string) string {
	if settingsProvider != nil {
		if val := settingsProvider.Get(key); val != "" {
			return val
		}
	}
	if envValue != "" && !deprecationWarned[envName] {
		slog.Warn("DEPRECATED: Using environment variable instead of database setting",
			"env", envName,
			"setting", key,
			"hint", "Configure this in admin panel Settings instead")
		deprecationWarned[envName] = true
	}
	return envValue
}

// getBoolSettingWithDeprecation checks DB first for bool settings
func getBoolSettingWithDeprecation(key string, envValue bool, envName string) bool {
	if settingsProvider != nil {
		if val := settingsProvider.Get(key); val != "" {
			return val == "true"
		}
	}
	if !deprecationWarned[envName] {
		slog.Warn("DEPRECATED: Using environment variable instead of database setting",
			"env", envName,
			"setting", key,
			"hint", "Configure this in admin panel Settings instead")
		deprecationWarned[envName] = true
	}
	return envValue
}

// getIntSettingWithDeprecation checks DB first for int settings
func getIntSettingWithDeprecation(key string, envValue int, envName string) int {
	if settingsProvider != nil {
		if val := settingsProvider.GetInt(key, -1); val != -1 {
			return val
		}
	}
	if !deprecationWarned[envName] {
		slog.Warn("DEPRECATED: Using environment variable instead of database setting",
			"env", envName,
			"setting", key,
			"hint", "Configure this in admin panel Settings instead")
		deprecationWarned[envName] = true
	}
	return envValue
}

func RemnawaveTag() string {
	return conf.remnawaveTag
}

func TrialRemnawaveTag() string {
	// Check DB setting first
	if settingsProvider != nil {
		if val := settingsProvider.Get("trial_remnawave_tag"); val != "" {
			return val
		}
	}
	// Fallback to env
	if conf.trialRemnawaveTag != "" {
		if !deprecationWarned["TRIAL_REMNAWAVE_TAG"] {
			slog.Warn("DEPRECATED: Using environment variable instead of database setting",
				"env", "TRIAL_REMNAWAVE_TAG",
				"setting", "trial_remnawave_tag",
				"hint", "Configure this in admin panel Settings instead")
			deprecationWarned["TRIAL_REMNAWAVE_TAG"] = true
		}
		return conf.trialRemnawaveTag
	}
	return conf.remnawaveTag
}
func BotAdminURL() string {
	return conf.botAdminURL
}

func DefaultLanguage() string {
	return conf.defaultLanguage
}
func GetTributeWebHookUrl() string {
	return getSettingWithDeprecation("tribute_webhook_url", conf.tributeWebhookUrl, "TRIBUTE_WEBHOOK_URL")
}

func GetTributeAPIKey() string {
	return getSettingWithDeprecation("tribute_api_key", conf.tributeAPIKey, "TRIBUTE_API_KEY")
}

func GetTributePaymentUrl() string {
	return getSettingWithDeprecation("tribute_payment_url", conf.tributePaymentUrl, "TRIBUTE_PAYMENT_URL")
}

func GetReferralDays() int {
	return conf.referralDays
}

func GetMiniAppURL() string {
	return getSettingWithDeprecation("mini_app_url", conf.miniApp, "MINI_APP_URL")
}

func SquadUUIDs() map[uuid.UUID]uuid.UUID {
	return conf.squadUUIDs
}

func GetBlockedTelegramIds() map[int64]bool {
	return conf.blockedTelegramIds
}

func GetWhitelistedTelegramIds() map[int64]bool {
	return conf.whitelistedTelegramIds
}

func TrialInternalSquads() map[uuid.UUID]uuid.UUID {
	// Check DB setting first
	if settingsProvider != nil {
		if val := settingsProvider.Get("trial_internal_squads"); val != "" {
			return parseUUIDsFromString(val)
		}
	}
	// Fallback to env
	if conf.trialInternalSquads != nil && len(conf.trialInternalSquads) > 0 {
		if !deprecationWarned["TRIAL_INTERNAL_SQUADS"] {
			slog.Warn("DEPRECATED: Using environment variable instead of database setting",
				"env", "TRIAL_INTERNAL_SQUADS",
				"setting", "trial_internal_squads",
				"hint", "Configure this in admin panel Settings instead")
			deprecationWarned["TRIAL_INTERNAL_SQUADS"] = true
		}
		return conf.trialInternalSquads
	}
	return conf.squadUUIDs
}

func TrialExternalSquadUUID() uuid.UUID {
	// Check DB setting first
	if settingsProvider != nil {
		if val := settingsProvider.Get("trial_external_squad_uuid"); val != "" {
			if parsed, err := uuid.Parse(val); err == nil {
				return parsed
			}
		}
	}
	// Fallback to env
	if conf.trialExternalSquadUUID != uuid.Nil {
		if !deprecationWarned["TRIAL_EXTERNAL_SQUAD_UUID"] {
			slog.Warn("DEPRECATED: Using environment variable instead of database setting",
				"env", "TRIAL_EXTERNAL_SQUAD_UUID",
				"setting", "trial_external_squad_uuid",
				"hint", "Configure this in admin panel Settings instead")
			deprecationWarned["TRIAL_EXTERNAL_SQUAD_UUID"] = true
		}
		return conf.trialExternalSquadUUID
	}
	return conf.externalSquadUUID
}

// parseUUIDsFromString parses comma-separated UUIDs into a map
func parseUUIDsFromString(s string) map[uuid.UUID]uuid.UUID {
	result := make(map[uuid.UUID]uuid.UUID)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if parsed, err := uuid.Parse(part); err == nil {
			result[parsed] = parsed
		}
	}
	return result
}

func TrialTrafficLimit() int {
	return getIntSettingWithDeprecation("trial_traffic_limit", conf.trialTrafficLimit, "TRIAL_TRAFFIC_LIMIT") * bytesInGigabyte
}

func TrialDays() int {
	return getIntSettingWithDeprecation("trial_days", conf.trialDays, "TRIAL_DAYS")
}

func FeedbackURL() string {
	return getSettingWithDeprecation("feedback_url", conf.feedbackURL, "FEEDBACK_URL")
}

func ChannelURL() string {
	return getSettingWithDeprecation("channel_url", conf.channelURL, "CHANNEL_URL")
}

func ServerStatusURL() string {
	return getSettingWithDeprecation("server_status_url", conf.serverStatusURL, "SERVER_STATUS_URL")
}

func SupportURL() string {
	return getSettingWithDeprecation("support_url", conf.supportURL, "SUPPORT_URL")
}

func TosURL() string {
	return conf.tosURL
}

func YookasaEmail() string {
	return getSettingWithDeprecation("yookasa_email", conf.yookasaEmail, "YOOKASA_EMAIL")
}

func Price1() int {
	return conf.price1
}

func Price3() int {
	return conf.price3
}

func Price6() int {
	return conf.price6
}

func Price12() int {
	return conf.price12
}

func DaysInMonth() int {
	return conf.daysInMonth
}

func ExternalSquadUUID() uuid.UUID {
	return conf.externalSquadUUID
}

func Price(month int) int {
	switch month {
	case 1:
		return conf.price1
	case 3:
		return conf.price3
	case 6:
		return conf.price6
	case 12:
		return conf.price12
	default:
		return conf.price1
	}
}

func StarsPrice(month int) int {
	switch month {
	case 1:
		return conf.starsPrice1
	case 3:
		return conf.starsPrice3
	case 6:
		return conf.starsPrice6
	case 12:
		return conf.starsPrice12
	default:
		return conf.starsPrice1
	}
}
func TelegramToken() string {
	return conf.telegramToken
}
func RemnawaveUrl() string {
	return conf.remnawaveUrl
}
func DadaBaseUrl() string {
	return conf.databaseURL
}
func RemnawaveToken() string {
	return conf.remnawaveToken
}
func RemnawaveMode() string {
	return conf.remnawaveMode
}
func CryptoPayUrl() string {
	return getSettingWithDeprecation("crypto_pay_url", conf.cryptoPayURL, "CRYPTO_PAY_URL")
}

func CryptoPayToken() string {
	return getSettingWithDeprecation("crypto_pay_token", conf.cryptoPayToken, "CRYPTO_PAY_TOKEN")
}
func BotURL() string {
	return conf.botURL
}
func SetBotURL(botURL string) {
	conf.botURL = botURL
}
func YookasaUrl() string {
	return getSettingWithDeprecation("yookasa_url", conf.yookasaURL, "YOOKASA_URL")
}

func YookasaShopId() string {
	return getSettingWithDeprecation("yookasa_shop_id", conf.yookasaShopId, "YOOKASA_SHOP_ID")
}

func YookasaSecretKey() string {
	return getSettingWithDeprecation("yookasa_secret_key", conf.yookasaSecretKey, "YOOKASA_SECRET_KEY")
}
func TrafficLimit() int {
	return conf.trafficLimit * bytesInGigabyte
}

func IsCryptoPayEnabled() bool {
	return getBoolSettingWithDeprecation("crypto_pay_enabled", conf.isCryptoEnabled, "CRYPTO_PAY_ENABLED")
}

func IsYookasaEnabled() bool {
	return getBoolSettingWithDeprecation("yookasa_enabled", conf.isYookasaEnabled, "YOOKASA_ENABLED")
}

func IsTelegramStarsEnabled() bool {
	return getBoolSettingWithDeprecation("telegram_stars_enabled", conf.isTelegramStarsEnabled, "TELEGRAM_STARS_ENABLED")
}

func RequirePaidPurchaseForStars() bool {
	return getBoolSettingWithDeprecation("require_paid_purchase_for_stars", conf.requirePaidPurchaseForStars, "REQUIRE_PAID_PURCHASE_FOR_STARS")
}

func GetAdminTelegramId() int64 {
	return conf.adminTelegramId
}

func GetHealthCheckPort() int {
	return conf.healthCheckPort
}

func IsWepAppLinkEnabled() bool {
	return conf.isWebAppLinkEnabled
}

func RemnawaveHeaders() map[string]string {
	return conf.remnawaveHeaders
}

func TrialTrafficLimitResetStrategy() string {
	return conf.trialTrafficLimitResetStrategy
}

func TrafficLimitResetStrategy() string {
	return conf.trafficLimitResetStrategy
}

const bytesInGigabyte = 1073741824

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Panicf("env %q not set", key)
	}
	return v
}

func mustEnvInt(key string) int {
	v := mustEnv(key)
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Panicf("invalid int in %q: %v", key, err)
	}
	return i
}

func envIntDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Panicf("invalid int in %q: %v", key, err)
	}
	return i
}

func envStringDefault(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envBool(key string) bool {
	return os.Getenv(key) == "true"
}

func InitConfig() {
	if os.Getenv("DISABLE_ENV_FILE") != "true" {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env loaded:", err)
		}
	}
	var err error
	conf.adminTelegramId, err = strconv.ParseInt(os.Getenv("ADMIN_TELEGRAM_ID"), 10, 64)
	if err != nil {
		panic("ADMIN_TELEGRAM_ID .env variable not set")
	}

	conf.telegramToken = mustEnv("TELEGRAM_TOKEN")

	conf.isWebAppLinkEnabled = func() bool {
		isWebAppLinkEnabled := os.Getenv("IS_WEB_APP_LINK") == "true"
		return isWebAppLinkEnabled
	}()

	conf.miniApp = envStringDefault("MINI_APP_URL", "")

	conf.remnawaveTag = envStringDefault("REMNAWAVE_TAG", "")

	conf.trialRemnawaveTag = envStringDefault("TRIAL_REMNAWAVE_TAG", "")

	conf.trialTrafficLimitResetStrategy = envStringDefault("TRIAL_TRAFFIC_LIMIT_RESET_STRATEGY", "MONTH")
	conf.trafficLimitResetStrategy = envStringDefault("TRAFFIC_LIMIT_RESET_STRATEGY", "MONTH")

	conf.defaultLanguage = envStringDefault("DEFAULT_LANGUAGE", "ru")

	conf.daysInMonth = envIntDefault("DAYS_IN_MONTH", 30)

	externalSquadUUIDStr := os.Getenv("EXTERNAL_SQUAD_UUID")
	if externalSquadUUIDStr != "" {
		parsedUUID, err := uuid.Parse(externalSquadUUIDStr)
		if err != nil {
			panic(fmt.Sprintf("invalid EXTERNAL_SQUAD_UUID format: %v", err))
		}
		conf.externalSquadUUID = parsedUUID
	} else {
		conf.externalSquadUUID = uuid.Nil
	}

	conf.trialTrafficLimit = envIntDefault("TRIAL_TRAFFIC_LIMIT", 0)

	conf.healthCheckPort = envIntDefault("HEALTH_CHECK_PORT", 8080)

	conf.trialDays = envIntDefault("TRIAL_DAYS", 0)

	conf.enableAutoPayment = envBool("ENABLE_AUTO_PAYMENT")

	conf.price1 = envIntDefault("PRICE_1", 0)
	conf.price3 = envIntDefault("PRICE_3", 0)
	conf.price6 = envIntDefault("PRICE_6", 0)
	conf.price12 = envIntDefault("PRICE_12", 0)

	conf.isTelegramStarsEnabled = envBool("TELEGRAM_STARS_ENABLED")
	if conf.isTelegramStarsEnabled {
		conf.starsPrice1 = envIntDefault("STARS_PRICE_1", conf.price1)
		conf.starsPrice3 = envIntDefault("STARS_PRICE_3", conf.price3)
		conf.starsPrice6 = envIntDefault("STARS_PRICE_6", conf.price6)
		conf.starsPrice12 = envIntDefault("STARS_PRICE_12", conf.price12)

	}

	conf.requirePaidPurchaseForStars = envBool("REQUIRE_PAID_PURCHASE_FOR_STARS")

	conf.remnawaveUrl = mustEnv("REMNAWAVE_URL")

	conf.remnawaveMode = func() string {
		v := os.Getenv("REMNAWAVE_MODE")
		if v != "" {
			if v != "remote" && v != "local" {
				panic("REMNAWAVE_MODE .env variable must be either 'remote' or 'local'")
			} else {
				return v
			}
		} else {
			return "remote"
		}
	}()

	conf.remnawaveToken = mustEnv("REMNAWAVE_TOKEN")

	conf.databaseURL = mustEnv("DATABASE_URL")

	conf.isCryptoEnabled = envBool("CRYPTO_PAY_ENABLED")
	conf.cryptoPayURL = envStringDefault("CRYPTO_PAY_URL", "")
	conf.cryptoPayToken = envStringDefault("CRYPTO_PAY_TOKEN", "")

	conf.isYookasaEnabled = envBool("YOOKASA_ENABLED")
	conf.yookasaURL = envStringDefault("YOOKASA_URL", "")
	conf.yookasaShopId = envStringDefault("YOOKASA_SHOP_ID", "")
	conf.yookasaSecretKey = envStringDefault("YOOKASA_SECRET_KEY", "")
	conf.yookasaEmail = envStringDefault("YOOKASA_EMAIL", "")

	conf.trafficLimit = envIntDefault("TRAFFIC_LIMIT", 0)
	conf.referralDays = envIntDefault("REFERRAL_DAYS", 0)

	conf.serverStatusURL = os.Getenv("SERVER_STATUS_URL")
	conf.supportURL = os.Getenv("SUPPORT_URL")
	conf.feedbackURL = os.Getenv("FEEDBACK_URL")
	conf.channelURL = os.Getenv("CHANNEL_URL")
	conf.tosURL = os.Getenv("TOS_URL")

	conf.squadUUIDs = func() map[uuid.UUID]uuid.UUID {
		v := os.Getenv("SQUAD_UUIDS")
		if v != "" {
			uuids := strings.Split(v, ",")
			var inboundsMap = make(map[uuid.UUID]uuid.UUID)
			for _, value := range uuids {
				uuid, err := uuid.Parse(value)
				if err != nil {
					panic(err)
				}
				inboundsMap[uuid] = uuid
			}
			slog.Info("Loaded squad UUIDs", "uuids", uuids)
			return inboundsMap
		} else {
			slog.Info("No squad UUIDs specified, all will be used")
			return map[uuid.UUID]uuid.UUID{}
		}
	}()

	conf.tributeWebhookUrl = envStringDefault("TRIBUTE_WEBHOOK_URL", "")
	conf.tributeAPIKey = envStringDefault("TRIBUTE_API_KEY", "")
	conf.tributePaymentUrl = envStringDefault("TRIBUTE_PAYMENT_URL", "")

	conf.blockedTelegramIds = func() map[int64]bool {
		v := os.Getenv("BLOCKED_TELEGRAM_IDS")
		if v != "" {
			ids := strings.Split(v, ",")
			var blockedMap = make(map[int64]bool)
			for _, idStr := range ids {
				id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
				if err != nil {
					panic(fmt.Sprintf("invalid telegram ID in BLOCKED_TELEGRAM_IDS: %v", err))
				}
				blockedMap[id] = true
			}
			slog.Info("Loaded blocked telegram IDs", "count", len(blockedMap))
			return blockedMap
		} else {
			slog.Info("No blocked telegram IDs specified")
			return map[int64]bool{}
		}
	}()

	conf.whitelistedTelegramIds = func() map[int64]bool {
		v := os.Getenv("WHITELISTED_TELEGRAM_IDS")
		if v != "" {
			ids := strings.Split(v, ",")
			var whitelistedMap = make(map[int64]bool)
			for _, idStr := range ids {
				id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
				if err != nil {
					panic(fmt.Sprintf("invalid telegram ID in WHITELISTED_TELEGRAM_IDS: %v", err))
				}
				whitelistedMap[id] = true
			}
			slog.Info("Loaded whitelisted telegram IDs", "count", len(whitelistedMap))
			return whitelistedMap
		} else {
			slog.Info("No whitelisted telegram IDs specified")
			return map[int64]bool{}
		}
	}()

	conf.trialInternalSquads = func() map[uuid.UUID]uuid.UUID {
		v := os.Getenv("TRIAL_INTERNAL_SQUADS")
		if v != "" {
			uuids := strings.Split(v, ",")
			var trialSquadsMap = make(map[uuid.UUID]uuid.UUID)
			for _, value := range uuids {
				parsedUUID, err := uuid.Parse(strings.TrimSpace(value))
				if err != nil {
					panic(fmt.Sprintf("invalid UUID in TRIAL_INTERNAL_SQUADS: %v", err))
				}
				trialSquadsMap[parsedUUID] = parsedUUID
			}
			slog.Info("Loaded trial internal squad UUIDs", "uuids", uuids)
			return trialSquadsMap
		} else {
			slog.Info("No trial internal squads specified, will use regular SQUAD_UUIDS for trial users")
			return map[uuid.UUID]uuid.UUID{}
		}
	}()

	trialExternalSquadUUIDStr := os.Getenv("TRIAL_EXTERNAL_SQUAD_UUID")
	if trialExternalSquadUUIDStr != "" {
		parsedUUID, err := uuid.Parse(trialExternalSquadUUIDStr)
		if err != nil {
			panic(fmt.Sprintf("invalid TRIAL_EXTERNAL_SQUAD_UUID format: %v", err))
		}
		conf.trialExternalSquadUUID = parsedUUID
		slog.Info("Loaded trial external squad UUID", "uuid", trialExternalSquadUUIDStr)
	} else {
		conf.trialExternalSquadUUID = uuid.Nil
		slog.Info("No trial external squad specified, will use regular EXTERNAL_SQUAD_UUID for trial users")
	}

	conf.botAdminURL = os.Getenv("BOT_ADMIN_URL")
	if conf.botAdminURL == "" {
		panic(fmt.Sprintf("BOT_ADMIN_URL environment variable not set"))
	}

	conf.remnawaveHeaders = func() map[string]string {
		v := os.Getenv("REMNAWAVE_HEADERS")
		if v != "" {
			headers := make(map[string]string)
			pairs := strings.Split(v, ";")
			for _, pair := range pairs {
				parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if key != "" && value != "" {
						headers[key] = value
					}
				}
			}
			if len(headers) > 0 {
				slog.Info("Loaded remnawave headers", "count", len(headers))
				return headers
			}
		}
		return map[string]string{}
	}()
}
