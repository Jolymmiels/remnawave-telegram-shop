# Remnawave Telegram Shop

[![Stars](https://img.shields.io/github/stars/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/stargazers)
[![Forks](https://img.shields.io/github/forks/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/network/members)
[![Issues](https://img.shields.io/github/issues/Jolymmiels/remnawave-telegram-shop.svg)](https://github.com/Jolymmiels/remnawave-telegram-shop/issues)

## Description

A Telegram bot for selling subscriptions with integration to Remnawave (https://remna.st/). This service allows users to
purchase and manage subscriptions through Telegram with multiple payment system options.

- [remnawave-api-go](https://github.com/Jolymmiles/remnawave-api-go)
- [Documentation](https://remnawave-telegram-shop-bot-doc.vercel.app/)

## Admin commands

- `/sync` - Poll users from remnawave and synchronize them with the database. Remove all users which not present in
  remnawave.

### Payment Systems

- [YooKassa API](https://yookassa.ru/developers/api)
- [CryptoPay API](https://help.crypt.bot/crypto-pay-api)
- Telegram Stars
- Tribute

## Features

- Purchase VPN subscriptions with different payment methods (bank cards, cryptocurrency)
- Multiple subscription plans (1, 3, 6, 12 months)
- Automated subscription management
- **Subscription Notifications**: The bot automatically sends notifications to users 3 days before their subscription
  expires, helping them avoid service interruption
- Multi-language support (Russian and English)
- **Selective Squad Assignment**: Configure specific squads to assign to users via UUID filtering
- All telegram message support HTML formatting https://core.telegram.org/bots/api#html-style
- Healthcheck - bot checking availability of db, panel.

## Version Support

| Remnawave     | Bot   |
|---------------|-------|
| 1.6           | 2.3.6 |
| 2.0.0 - 2.1.9 | 3.2.4 |
| 2.2.*         | 3.2.5 |
| 2.3.*         | 3.5.* |

## API

Web server start on port defined in .env via HEALTH_CHECK_PORT

- /healthcheck
- /${TRIBUTE_PAYMENT_URL} - webhook for tribute

## Environment Variables

The application requires the following environment variables to be set:

| Variable                 | Description                                                                                                                                |
|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------| 
| `PRICE_1`                | Price for 1 month                                                                                                                          |
| `PRICE_3`                | Price for 3 month                                                                                                                          |
| `PRICE_6`                | Price for 6 month                                                                                                                          |
| `PRICE_12`               | Price for 12 month                                                                                                                         |
| `DAYS_IN_MONTH`          | Days in month                                                                                                                              |
| `DEFAULT_LANGUAGE`       | Default language for bot messages (en or ru). Default: ru                                                                                   |
| `REMNAWAVE_TAG`          | Tag in remnawave                                                                                                                           |
| `TRIAL_REMNAWAVE_TAG`    | Tag to assign to trial users in Remnawave (optional, if not set, regular REMNAWAVE_TAG will be used)                                        |
| `HEALTH_CHECK_PORT`      | Server port                                                                                                                                |
| `IS_WEB_APP_LINK`        | If true, then sublink will be showed as webapp..                                                                                           |
| `REMNAWAVE_HEADERS`      | Additional headers for remnawave requests (format: key1:value1;key2:value2). Example: X-Api-Key:your_key;X-Custom:value (optional)       |
| `MINI_APP_URL`           | tg WEB APP URL. if empty not be used.                                                                                                      |
| `STARS_PRICE_1`          | Price in Stars for 1 month                                                                                                                 
| `STARS_PRICE_3`          | Price in Stars for 3 month                                                                                                                 
| `STARS_PRICE_6`          | Price in Stars for 6 month                                                                                                                 
| `STARS_PRICE_12`         | Price in Stars for 12 month                                                                                                                
| `REFERRAL_DAYS`          | Refferal days. if 0, then disabled.                                                                                                        |
| `TELEGRAM_TOKEN`         | Telegram Bot API token for bot functionality                                                                                               |
| `DATABASE_URL`           | PostgreSQL connection string                                                                                                               |
| `POSTGRES_USER`          | PostgreSQL username                                                                                                                        |
| `POSTGRES_PASSWORD`      | PostgreSQL password                                                                                                                        |
| `POSTGRES_DB`            | PostgreSQL database name                                                                                                                   |
| `REMNAWAVE_URL`          | Remnawave API URL                                                                                                                          |
| `REMNAWAVE_MODE`         | Remnawave mode (remote/local), default is remote. If local set – you can pass http://remnawave:3000 to REMNAWAVE_URL                       |
| `REMNAWAVE_TOKEN`        | Authentication token for Remnawave API                                                                                                     |
| `CRYPTO_PAY_ENABLED`     | Enable/disable CryptoPay payment method (true/false)                                                                                       |
| `CRYPTO_PAY_TOKEN`       | CryptoPay API token                                                                                                                        |
| `CRYPTO_PAY_URL`         | CryptoPay API URL                                                                                                                          |
| `YOOKASA_ENABLED`        | Enable/disable YooKassa payment method (true/false)                                                                                        |
| `YOOKASA_SECRET_KEY`     | YooKassa API secret key                                                                                                                    |
| `YOOKASA_SHOP_ID`        | YooKassa shop identifier                                                                                                                   |
| `YOOKASA_URL`            | YooKassa API URL                                                                                                                           |
| `YOOKASA_EMAIL`          | Email address associated with YooKassa account                                                                                             |
| `TRAFFIC_LIMIT`          | Maximum allowed traffic in gb (0 to set unlimited)                                                                                         |
| `TELEGRAM_STARS_ENABLED` | Enable/disable Telegram Stars payment method (true/false)                                                                                  |
| `REQUIRE_PAID_PURCHASE_FOR_STARS` | Require successful cryptocurrency or card payment before allowing Telegram Stars (true/false). Default: false |
| `SERVER_STATUS_URL`      | URL to server status page (optional) - if not set, button will not be displayed                                                            |
| `SUPPORT_URL`            | URL to support chat or page (optional) - if not set, button will not be displayed                                                          |
| `FEEDBACK_URL`           | URL to feedback/reviews page (optional) - if not set, button will not be displayed                                                         |
| `CHANNEL_URL`            | URL to Telegram channel (optional) - if not set, button will not be displayed                                                              |
| `TOS_URL`                | URL to TOS (optional) - if not set, button will not be displayed                                                                           |
| `ADMIN_TELEGRAM_ID`      | Admin telegram id                                                                                                                          |
| `BLOCKED_TELEGRAM_IDS`   | Comma-separated list of Telegram IDs to block from accessing the bot (e.g., "123456789,987654321")                                         |
| `WHITELISTED_TELEGRAM_IDS` | Comma-separated list of Telegram IDs that bypass all suspicious user checks (e.g., "111111111,222222222,333333333")                      |
| `TRIAL_TRAFFIC_LIMIT`    | Maximum allowed traffic in gb for trial subscriptions                                                                                      |     
| `TRIAL_DAYS`             | Number of days for trial subscriptions. if 0 = disabled.                                                                                   |
| `TRAFFIC_LIMIT_RESET_STRATEGY` | Traffic limit reset strategy. Allowed values: DAY, WEEK, MONTH, NO_RESET. Default: MONTH. |
| `TRIAL_TRAFFIC_LIMIT_RESET_STRATEGY` | Traffic limit reset strategy for trial users. Allowed values: DAY, WEEK, MONTH, NO_RESET. Default: MONTH. |
| `TRIAL_INTERNAL_SQUADS`  | Comma-separated list of squad UUIDs to assign to trial users (optional, if not set, regular SQUAD_UUIDS will be used)                      |
| `TRIAL_EXTERNAL_SQUAD_UUID` | Single external squad UUID to assign to trial users during creation and updates (optional, if not set, regular EXTERNAL_SQUAD_UUID will be used) |
| `SQUAD_UUIDS`            | Comma-separated list of squad UUIDs to assign to users (e.g., "773db654-a8b2-413a-a50b-75c3536238fd,bc979bdd-f1fa-4d94-8a51-38a0f518a2a2") |
| `EXTERNAL_SQUAD_UUID`    | Single external squad UUID to assign to users during creation and updates (optional, e.g., "773db654-a8b2-413a-a50b-75c3536238fd")        |
| `TRIBUTE_WEBHOOK_URL`    | Path for webhook handler. Example: /example (https://www.uuidgenerator.net/version4)                                                       |
| `TRIBUTE_API_KEY`        | Api key, which can be obtained via settings in Tribute app.                                                                                |
| `TRIBUTE_PAYMENT_URL`    | You payment url for Tribute. (Subscription telegram link)                                                                                  |
| `TELEGRAM_PROXY_URL`     | Proxy URL for Telegram Bot API requests (optional, e.g., `socks5://host:port` or `http://host:port`)                                      |
| `MOYNALOG_PROXY_URL`     | Proxy URL for Moy Nalog API requests (optional, e.g., `socks5://host:port` or `http://host:port`)                                         |

## User Interface

The bot dynamically creates buttons based on available environment variables:

- Main buttons for purchasing and connecting to the VPN are always shown
- Additional buttons for Server Status, Support, Feedback, and Channel are only displayed if their corresponding URL
  environment variables are set

## Automated Notifications

The bot includes a notification system that runs daily at 16:00 UTC to check for expiring subscriptions:

- Users receive a notification 3 days before their subscription expires
- The notification includes the exact expiration date and a convenient button to renew the subscription
- Notifications are sent in the user's preferred language

## Squad Configuration

The bot supports selective squad assignment to users:

### Internal Squads (SQUAD_UUIDS)

- Configure specific squad UUIDs in the `SQUAD_UUIDS` environment variable (comma-separated)
- If specified, only squads with matching UUIDs will be assigned to new users
- If no squads match the specified UUIDs or the variable is empty, all available squads will be assigned
- This feature allows fine-grained control over which connection methods are available to users

### External Squad (EXTERNAL_SQUAD_UUID)

- Configure a single external squad UUID in the `EXTERNAL_SQUAD_UUID` environment variable
- When set, this external squad will be included in all user creation and update requests to the Remnawave API
- The UUID is validated and parsed during application startup; invalid formats will prevent the application from starting
- Leave empty to disable external squad assignment

### Trial Squad Configuration (TRIAL_INTERNAL_SQUADS and TRIAL_EXTERNAL_SQUAD_UUID)

Trial users can be assigned to different squads than regular paying users:

#### Trial Internal Squads (TRIAL_INTERNAL_SQUADS)

- Configure specific squad UUIDs for trial users in the `TRIAL_INTERNAL_SQUADS` environment variable (comma-separated)
- If specified, only trial-specific squads will be assigned to users activating trial subscriptions
- If empty or not set, trial users will be assigned to regular squads defined in `SQUAD_UUIDS` (fallback behavior)
- Example: `TRIAL_INTERNAL_SQUADS=773db654-a8b2-413a-a50b-75c3536238fd,bc979bdd-f1fa-4d94-8a51-38a0f518a2a2`

#### Trial External Squad (TRIAL_EXTERNAL_SQUAD_UUID)

- Configure a single external squad UUID for trial users in the `TRIAL_EXTERNAL_SQUAD_UUID` environment variable
- If specified, this external squad will be included in all trial user creation and update requests
- If empty or not set, trial users will use the regular squad defined in `EXTERNAL_SQUAD_UUID` (fallback behavior)
- Example: `TRIAL_EXTERNAL_SQUAD_UUID=773db654-a8b2-413a-a50b-75c3536238fd`

**Use Case:** Isolate trial users in separate squad(s) for monitoring, resource allocation, or specific feature testing

## Plugins and Dependencies

### Telegram Bot

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Go Telegram Bot API](https://github.com/go-telegram/bot)

### Database

- [PostgreSQL](https://www.postgresql.org/)
- [pgx - PostgreSQL Driver](https://github.com/jackc/pgx)

## Setup Instructions

1. Clone the repository

```bash
git clone https://github.com/Jolymmiels/remnawave-telegram-shop && cd remnawave-telegram-shop
```

2. Create a `.env` file in the root directory with all the environment variables listed above

```bash
mv .env.sample .env
```

3. Run the bot:

```bash
docker compose up -d
```

## Tribute payment setup instructions

> [!WARNING]
> To integrate with Tribute, you must have a public domain (e.g., `bot.example.com`) that points to your bot server.  
> Webhook and subscription setup will not work on a local address or IP — only via a domain with a valid SSL
> certificate.

### How the integration works

The bot supports subscription management via the Tribute service. When a user clicks the payment button, they are
redirected to the Tribute bot or payment page to complete the subscription. After successful payment, Tribute sends a
webhook to your server, and the bot activates the subscription for the user.

### Step-by-step setup guide

1. Getting started

* Create a channel;
* In the Tribute app, open "Channels and Groups" and add your channel;
* Create a new subscription;
* Obtain the subscription link (Subscription -> Links -> Telegram Link).

2. Configure environment variables in `.env`
    * Set the webhook path (e.g., `/tribute/webhook`):

    ```
    TRIBUTE_WEBHOOK_URL=/tribute/webhook
    ```

    * Set the API key from your Tribute settings:

    ```
    TRIBUTE_API_KEY=your_tribute_api_key
    ```

    * Paste the subscription link you got from Tribute:

    ```
    TRIBUTE_PAYMENT_URL=https://t.me/tribute/app?startapp=...
    ```

    * Specify the port the app will use:

    ```
    HEALTH_CHECK_PORT=82251
    ```

3. Restart bot

```bash
docker compose down && docker compose up -d
```

## How to change bot messages

Go to folder translations inside bot folder and change needed language.

## Update Instructions

1. Pull the latest Docker image:

```bash
docker compose pull
```

2. Restart the containers:

```bash
docker compose down && docker compose up -d
```

## Reverse Proxy Configuration

If you are not using ngrok from `docker-compose.yml`, you need to set up a reverse proxy to forward requests to the bot.

<details>
<summary>Traefik Configuration</summary>

```yaml
http:
  routers:
    remnawave-telegram-shop:
      rule: "Host(`bot.example.com`)"
      entrypoints:
        - http
      middlewares:
        - redirect-to-https
      service: remnawave-telegram-shop

    remnawave-telegram-shop-secure:
      rule: "Host(`bot.example.com`)"
      entrypoints:
        - https
      tls:
        certResolver: letsencrypt
      service: remnawave-telegram-shop

  middlewares:
    redirect-to-https:
      redirectScheme:
        scheme: https

  services:
    remnawave-telegram-shop:
      loadBalancer:
        servers:
          - url: "http://bot:82251"
```

</details>

## Donations

If you appreciate this project and want to help keep it running (and fuel those caffeine-fueled coding marathons),
consider donating. Your support helps drive future updates and improvements.

**Donation Methods:**

- **Bep20 USDT:** `0x843ff093ec36d45db5e21bca80b5bc315d03a433`

- **SOL Solana:** `8eXDeyZAqnq8nENtooMNSVExWJ1eq6J9nDDmjC9NJza7`

- **TRC20 USDT:** `TW7f4B4isWCn1QHdnrNG8CZ66cMffAMgN3`

- **TON USDT:** `UQALE1A_KMyWAYKuIugEFUMC_H0bTOMmrEyMKBUJ8oyRUCug`
