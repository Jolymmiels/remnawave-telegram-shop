package config

import (
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type config struct {
	telegramToken                                             string
	price1, price3, price6, price12                           int
	starsPrice1, starsPrice3, starsPrice6, starsPrice12       int
	remnawaveUrl, remnawaveToken, remnawaveMode               string
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
	adminTelegramIds                                          map[int64]struct{}
	trialDays                                                 int
	inboundUUIDs                                              map[uuid.UUID]uuid.UUID
	referralDays                                              int
	miniApp                                                   string
	enableAutoPayment                                         bool
	healthCheckPort                                           int
	tributeWebhookUrl, tributeAPIKey, tributePaymentUrl       string
	isWebAppLinkEnabled                                       bool
	xApiKey                                                   string
}

var conf config

func GetTributeWebHookUrl() string {
	return conf.tributeWebhookUrl
}
func GetTributeAPIKey() string {
	return conf.tributeAPIKey
}

func GetTributePaymentUrl() string {
	return conf.tributePaymentUrl
}

func GetReferralDays() int {
	return conf.referralDays
}

func GetMiniAppURL() string {
	return conf.miniApp
}

func InboundUUIDs() map[uuid.UUID]uuid.UUID {
	return conf.inboundUUIDs
}

func TrialTrafficLimit() int {
	return conf.trialTrafficLimit * bytesInGigabyte
}

func TrialDays() int {
	return conf.trialDays
}
func FeedbackURL() string {
	return conf.feedbackURL
}

func ChannelURL() string {
	return conf.channelURL
}

func ServerStatusURL() string {
	return conf.serverStatusURL
}

func SupportURL() string {
	return conf.supportURL
}

func TosURL() string {
	return conf.tosURL
}

func YookasaEmail() string {
	return conf.yookasaEmail
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
	return conf.cryptoPayURL
}
func CryptoPayToken() string {
	return conf.cryptoPayToken
}
func BotURL() string {
	return conf.botURL
}
func SetBotURL(botURL string) {
	conf.botURL = botURL
}
func YookasaUrl() string {
	return conf.yookasaURL
}
func YookasaShopId() string {
	return conf.yookasaShopId
}
func YookasaSecretKey() string {
	return conf.yookasaSecretKey
}
func TrafficLimit() int {
	return conf.trafficLimit * bytesInGigabyte
}

func IsCryptoPayEnabled() bool {
	return conf.isCryptoEnabled
}

func IsYookasaEnabled() bool {
	return conf.isYookasaEnabled
}

func IsTelegramStarsEnabled() bool {
	return conf.isTelegramStarsEnabled
}

func IsAdmin(id int64) bool {
	if conf.adminTelegramIds == nil {
		return false
	}
	_, ok := conf.adminTelegramIds[id]
	return ok
}

func GetAdminTelegramIds() []int64 {
	ids := make([]int64, 0, len(conf.adminTelegramIds))
	for id := range conf.adminTelegramIds {
		ids = append(ids, id)
	}
	return ids
}

func GetHealthCheckPort() int {
	return conf.healthCheckPort
}

func IsWepAppLinkEnabled() bool {
	return conf.isWebAppLinkEnabled
}

func GetXApiKey() string {
	return conf.xApiKey
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

func envBool(key string) bool {
	return os.Getenv(key) == "true"
}

func InitConfig() {
	if os.Getenv("DISABLE_ENV_FILE") != "true" {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env loaded:", err)
		}
	}
	conf.adminTelegramIds = make(map[int64]struct{})
	idsStr := os.Getenv("ADMIN_TELEGRAM_IDS")
	if idsStr == "" {
		log.Panic("ADMIN_TELEGRAM_IDS .env variable not set")
	}
	for _, idStr := range strings.Split(idsStr, ",") {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Panicf("invalid admin id %s: %v", idStr, err)
		}
		conf.adminTelegramIds[id] = struct{}{}
	}

	conf.telegramToken = mustEnv("TELEGRAM_TOKEN")

	conf.xApiKey = os.Getenv("X_API_KEY")

	conf.isWebAppLinkEnabled = func() bool {
		isWebAppLinkEnabled := os.Getenv("IS_WEB_APP_LINK") == "true"
		return isWebAppLinkEnabled
	}()

	conf.miniApp = func() string {
		v := os.Getenv("MINI_APP_URL")
		if v != "" {
			return v
		} else {
			return ""
		}
	}()

	conf.trialTrafficLimit = mustEnvInt("TRIAL_TRAFFIC_LIMIT")

	conf.healthCheckPort = envIntDefault("HEALTH_CHECK_PORT", 8080)

	conf.trialDays = mustEnvInt("TRIAL_DAYS")

	conf.enableAutoPayment = envBool("ENABLE_AUTO_PAYMENT")

	conf.price1 = mustEnvInt("PRICE_1")
	conf.price3 = mustEnvInt("PRICE_3")
	conf.price6 = mustEnvInt("PRICE_6")
	conf.price12 = mustEnvInt("PRICE_12")

	conf.isTelegramStarsEnabled = envBool("TELEGRAM_STARS_ENABLED")
	if conf.isTelegramStarsEnabled {
		conf.starsPrice1 = envIntDefault("STARS_PRICE_1", conf.price1)
		conf.starsPrice3 = envIntDefault("STARS_PRICE_3", conf.price3)
		conf.starsPrice6 = envIntDefault("STARS_PRICE_6", conf.price6)
		conf.starsPrice12 = envIntDefault("STARS_PRICE_12", conf.price12)

	}

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
	if conf.isCryptoEnabled {
		conf.cryptoPayURL = mustEnv("CRYPTO_PAY_URL")
		conf.cryptoPayToken = mustEnv("CRYPTO_PAY_TOKEN")
	}

	conf.isYookasaEnabled = envBool("YOOKASA_ENABLED")
	if conf.isYookasaEnabled {
		conf.yookasaURL = mustEnv("YOOKASA_URL")
		conf.yookasaShopId = mustEnv("YOOKASA_SHOP_ID")
		conf.yookasaSecretKey = mustEnv("YOOKASA_SECRET_KEY")
		conf.yookasaEmail = mustEnv("YOOKASA_EMAIL")
	}

	conf.trafficLimit = mustEnvInt("TRAFFIC_LIMIT")
	conf.referralDays = mustEnvInt("REFERRAL_DAYS")

	conf.serverStatusURL = os.Getenv("SERVER_STATUS_URL")
	conf.supportURL = os.Getenv("SUPPORT_URL")
	conf.feedbackURL = os.Getenv("FEEDBACK_URL")
	conf.channelURL = os.Getenv("CHANNEL_URL")
	conf.tosURL = os.Getenv("TOS_URL")

	conf.inboundUUIDs = func() map[uuid.UUID]uuid.UUID {
		v := os.Getenv("INBOUND_UUIDS")
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
			slog.Info("Loaded inbound UUIDs", "uuids", uuids)
			return inboundsMap
		} else {
			slog.Info("No inbound UUIDs specified, all will be used")
			return map[uuid.UUID]uuid.UUID{}
		}
	}()

	conf.tributeWebhookUrl = os.Getenv("TRIBUTE_WEBHOOK_URL")
	if conf.tributeWebhookUrl != "" {
		conf.tributeAPIKey = mustEnv("TRIBUTE_API_KEY")
		conf.tributePaymentUrl = mustEnv("TRIBUTE_PAYMENT_URL")
	}
}
