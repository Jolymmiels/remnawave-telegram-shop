package config

import (
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type config struct {
	telegramToken          string
	price1                 int
	price3                 int
	price6                 int
	price12                int
	remnawaveUrl           string
	remnawaveToken         string
	remnawaveMode          string
	databaseURL            string
	cryptoPayURL           string
	cryptoPayToken         string
	botURL                 string
	yookasaURL             string
	yookasaShopId          string
	yookasaSecretKey       string
	yookasaEmail           string
	trafficLimit           int
	feedbackURL            string
	channelURL             string
	serverStatusURL        string
	supportURL             string
	tosURL                 string
	isYookasaEnabled       bool
	isCryptoEnabled        bool
	isTelegramStarsEnabled bool
	adminTelegramId        int64
	trialDays              int
	trialTrafficLimit      int
	inboundUUIDs           map[uuid.UUID]uuid.UUID
	referralDays           int
	miniApp                string
}

var conf config

func getEnvAsInt(name string, defaultValue int) int {
    valueStr := os.Getenv(name)
    if valueStr == "" {
        return defaultValue
    }
    value, err := strconv.Atoi(valueStr)
    if err != nil {
        slog.Warn("Invalid integer value for env var", "name", name, "value", valueStr)
        return defaultValue
    }
    return value
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
func Price1XTR() int {
    return getEnvAsInt("PRICE_1_XTR", 0)
}
func Price3XTR() int {
    return getEnvAsInt("PRICE_3_XTR", 0)
}
func Price6XTR() int {
    return getEnvAsInt("PRICE_6_XTR", 0)
}
func Price12XTR() int {
    return getEnvAsInt("PRICE_12_XTR", 0)
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

func GetAdminTelegramId() int64 {
	return conf.adminTelegramId
}

const bytesInGigabyte = 1073741824

func InitConfig() {
	err := godotenv.Load(".env")

	conf.adminTelegramId, err = strconv.ParseInt(os.Getenv("ADMIN_TELEGRAM_ID"), 10, 64)
	if err != nil {
		panic("ADMIN_TELEGRAM_ID .env variable not set")
	}

	conf.telegramToken = os.Getenv("TELEGRAM_TOKEN")
	if conf.telegramToken == "" {
		panic("TELEGRAM_TOKEN .env variable not set")
	}

	conf.miniApp = os.Getenv("MINI_APP_URL")
	if conf.miniApp == "" {
		conf.miniApp = ""
	}

	conf.trialTrafficLimit, err = strconv.Atoi(os.Getenv("TRIAL_TRAFFIC_LIMIT"))
	if err != nil {
		panic("TRIAL_TRAFFIC_LIMIT .env variable not set")
	}

	conf.trialDays, err = strconv.Atoi(os.Getenv("TRIAL_DAYS"))
	if err != nil {
		panic("TRIAL_DAYS .env variable not set")
	}

	strPrice := os.Getenv("PRICE_1")
	if strPrice == "" {
		panic("PRICE_1 .env variable not set")
	}
	price, err := strconv.Atoi(strPrice)
	if err != nil {
		panic(err)
	}
	conf.price1 = price

	strPrice3 := os.Getenv("PRICE_3")
	if strPrice3 == "" {
		panic("PRICE_3 .env variable not set")
	}
	price3, err := strconv.Atoi(strPrice3)
	if err != nil {
		panic(err)
	}
	conf.price3 = price3

	strPrice6 := os.Getenv("PRICE_6")
	if strPrice6 == "" {
		panic("PRICE_6 .env variable not set")
	}
	price6, err := strconv.Atoi(strPrice6)
	if err != nil {
		panic(err)
	}
	conf.price6 = price6

	strPrice12 := os.Getenv("PRICE_12")
	if strPrice12 == "" {
		panic("PRICE_12 .env variable not set")
	}
	price12, err := strconv.Atoi(strPrice12)
	if err != nil {
		panic(err)
	}
	conf.price12 = price12

	conf.remnawaveUrl = os.Getenv("REMNAWAVE_URL")
	if conf.remnawaveUrl == "" {
		panic("REMNAWAVE_URL .env variable not set")
	}

	conf.remnawaveMode = os.Getenv("REMNAWAVE_MODE")
	if conf.remnawaveMode == "" {
		conf.remnawaveMode = "remote"
	} else if conf.remnawaveMode != "remote" && conf.remnawaveMode != "local" {
		panic("REMNAWAVE_MODE .env variable must be either 'remote' or 'local'")
	}

	conf.remnawaveToken = os.Getenv("REMNAWAVE_TOKEN")
	if conf.remnawaveToken == "" {
		panic("REMNAWAVE_TOKEN .env variable not set")
	}

	conf.databaseURL = os.Getenv("DATABASE_URL")
	if conf.databaseURL == "" {
		panic("DADA_BASE_URL .env variable not set")
	}

	conf.isTelegramStarsEnabled = os.Getenv("TELEGRAM_STARS_ENABLED") == "true"

	conf.isCryptoEnabled = os.Getenv("CRYPTO_PAY_ENABLED") == "true"
	if conf.isCryptoEnabled {
		conf.cryptoPayURL = os.Getenv("CRYPTO_PAY_URL")
		if conf.cryptoPayURL == "" {
			panic("CRYPTO_PAY_URL .env variable not set")
		}
		conf.cryptoPayToken = os.Getenv("CRYPTO_PAY_TOKEN")
		if conf.cryptoPayToken == "" {
			panic("CRYPTO_PAY_TOKEN .env variable not set")
		}
	}

	conf.isYookasaEnabled = os.Getenv("YOOKASA_ENABLED") == "true"
	if conf.isYookasaEnabled {
		conf.yookasaURL = os.Getenv("YOOKASA_URL")
		conf.yookasaShopId = os.Getenv("YOOKASA_SHOP_ID")
		conf.yookasaSecretKey = os.Getenv("YOOKASA_SECRET_KEY")

		if conf.yookasaURL == "" || conf.yookasaShopId == "" || conf.yookasaSecretKey == "" {
			panic("YOOKASA_URL, YOOKASA_SHOP_ID, YOOKASA_SECRET_KEY .env variables not set")
		}

		conf.yookasaEmail = os.Getenv("YOOKASA_EMAIL")
		if conf.yookasaEmail == "" {
			panic("YOOKASA_EMAIL .env variable not set")
		}
	}

	strLimit := os.Getenv("TRAFFIC_LIMIT")
	if strLimit == "" {
		panic("TRAFFIC_LIMIT .env variable not set")
	}
	limit, err := strconv.Atoi(strLimit)
	if err != nil {
		panic(err)
	}
	conf.trafficLimit = limit

	conf.referralDays, err = strconv.Atoi(os.Getenv("REFERRAL_DAYS"))
	if err != nil {
		panic("REFERRAL_DAYS .env variable not set")
	}

	conf.serverStatusURL = os.Getenv("SERVER_STATUS_URL")
	conf.supportURL = os.Getenv("SUPPORT_URL")
	conf.feedbackURL = os.Getenv("FEEDBACK_URL")
	conf.channelURL = os.Getenv("CHANNEL_URL")
	conf.tosURL = os.Getenv("TOS_URL")

	inboundUUIDsStr := os.Getenv("INBOUND_UUIDS")
	if inboundUUIDsStr != "" {
		uuids := strings.Split(inboundUUIDsStr, ",")
		var inboundsMap = make(map[uuid.UUID]uuid.UUID)
		for _, value := range uuids {
			uuid, err := uuid.Parse(value)
			if err != nil {
				panic(err)
			}
			inboundsMap[uuid] = uuid
		}
		conf.inboundUUIDs = inboundsMap
		slog.Info("Loaded inbound UUIDs", "uuids", conf.inboundUUIDs)
	} else {
		conf.inboundUUIDs = map[uuid.UUID]uuid.UUID{}
		slog.Info("No inbound UUIDs specified, all will be used")
	}
}
