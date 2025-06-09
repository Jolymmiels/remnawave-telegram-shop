# Remnawave Telegram Shop
[![Stars](https://img.shields.io/github/stars/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/stargazers)
[![Forks](https://img.shields.io/github/forks/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/network/members)
[![Issues](https://img.shields.io/github/issues/Jolymmiels/remnawave-telegram-shop.svg)](https://github.com/Jolymmiels/remnawave-telegram-shop/issues)
## Description

A Telegram bot for selling subscriptions with integration to Remnawave (https://remna.st/). This service allows users to
purchase and manage subscriptions through Telegram with multiple payment system options.

- [remnawave-api-go](https://github.com/Jolymmiles/remnawave-api-go)

## Admin commands

- `/sync` - Poll users from remnawave and synchronize them with the database. Remove all users which not present in
  remnawave.

## Features

- Purchase VPN subscriptions with different payment methods (bank cards, cryptocurrency)
- Multiple subscription plans (1, 3, 6, 12 months)
- Automated subscription management
- **Subscription Notifications**: The bot automatically sends notifications to users 3 days before their subscription
  expires, helping them avoid service interruption
- Multi-language support (Russian and English)
- **Selective Inbound Assignment**: Configure specific inbounds to assign to users via UUID filtering
- All telegram message support HTML formatting https://core.telegram.org/bots/api#html-style
## Environment Variables

The application requires the following environment variables to be set:

| Variable                 | Description                                                                                                                                  |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------| 
| `PRICE_1`                | Price for 1 month                                                                                                                            |
| `PRICE_3`                | Price for 3 month                                                                                                                            |
| `PRICE_6`                | Price for 6 month                                                                                                                            |
| `MINI_APP_URL`           | tg WEB APP URL. if empty not be used.                                                                                                        |
| `PRICE_12`               | Price for 12 month                                                                                                                           |
| `PRICE_1_XTR             | Starts price for 1 month                                                                                                                     |
| `PRICE_3_XTR`            | Starts price for 3 month                                                                                                                     |
| `PRICE_6_XTR`            | Starts price for 6 month                                                                                                                     |
| `PRICE_12_XTR`           | Starts price for 6 month                                                                                                                     |
| `REFERRAL_DAYS`          | Refferal days. if 0, then disabled.                                                                                                          |
| `TELEGRAM_TOKEN`         | Telegram Bot API token for bot functionality                                                                                                 |
| `DATABASE_URL`           | PostgreSQL connection string                                                                                                                 |
| `POSTGRES_USER`          | PostgreSQL username                                                                                                                          |
| `POSTGRES_PASSWORD`      | PostgreSQL password                                                                                                                          |
| `POSTGRES_DB`            | PostgreSQL database name                                                                                                                     |
| `REMNAWAVE_URL`          | Remnawave API URL                                                                                                                            |
| `REMNAWAVE_MODE`         | Remnawave mode (remote/local), default is remote. If local set – you can pass http://remnawave:3000 to REMNAWAVE_URL                         |
| `REMNAWAVE_TOKEN`        | Authentication token for Remnawave API                                                                                                       |
| `CRYPTO_PAY_ENABLED`     | Enable/disable CryptoPay payment method (true/false)                                                                                         |
| `CRYPTO_PAY_TOKEN`       | CryptoPay API token                                                                                                                          |
| `CRYPTO_PAY_URL`         | CryptoPay API URL                                                                                                                            |
| `YOOKASA_ENABLED`        | Enable/disable YooKassa payment method (true/false)                                                                                          |
| `YOOKASA_SECRET_KEY`     | YooKassa API secret key                                                                                                                      |
| `YOOKASA_SHOP_ID`        | YooKassa shop identifier                                                                                                                     |
| `YOOKASA_URL`            | YooKassa API URL                                                                                                                             |
| `YOOKASA_EMAIL`          | Email address associated with YooKassa account                                                                                               |
| `TRAFFIC_LIMIT`          | Maximum allowed traffic in gb (0 to set unlimited)                                                                                           |
| `TELEGRAM_STARS_ENABLED` | Enable/disable Telegram Stars payment method (true/false)                                                                                    |
| `SERVER_STATUS_URL`      | URL to server status page (optional) - if not set, button will not be displayed                                                              |
| `SUPPORT_URL`            | URL to support chat or page (optional) - if not set, button will not be displayed                                                            |
| `FEEDBACK_URL`           | URL to feedback/reviews page (optional) - if not set, button will not be displayed                                                           |
| `CHANNEL_URL`            | URL to Telegram channel (optional) - if not set, button will not be displayed                                                                |
| `ADMIN_TELEGRAM_ID`      | Admin telegram id                                                                                                                            |
| `TRIAL_TRAFFIC_LIMIT`    | Maximum allowed traffic in gb for trial subscriptions                                                                                        |     
| `TRIAL_DAYS`             | Number of days for trial subscriptions. if 0 = disabled.                                                                                     |
| `INBOUND_UUIDS`          | Comma-separated list of inbound UUIDs to assign to users (e.g., "773db654-a8b2-413a-a50b-75c3536238fd,bc979bdd-f1fa-4d94-8a51-38a0f518a2a2") |

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

## Inbound Configuration

The bot supports selective inbound assignment to users:

- Configure specific inbound UUIDs in the `INBOUND_UUIDS` environment variable (comma-separated)
- If specified, only inbounds with matching UUIDs will be assigned to new users
- If no inbounds match the specified UUIDs or the variable is empty, all available inbounds will be assigned
- This feature allows fine-grained control over which connection methods are available to users

## Plugins and Dependencies

### Telegram Bot

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Go Telegram Bot API](https://github.com/go-telegram/bot)

### Database

- [PostgreSQL](https://www.postgresql.org/)
- [pgx - PostgreSQL Driver](https://github.com/jackc/pgx)

### Payment Systems

- [YooKassa API](https://yookassa.ru/developers/api)
- [CryptoPay API](https://help.crypt.bot/crypto-pay-api)
- Telegram Stars

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

## Donations

If you appreciate this project and want to help keep it running (and fuel those caffeine-fueled coding marathons),
consider donating. Your support helps drive future updates and improvements.

**Donation Methods:**

- **Bep20 USDT:** `0x4D1ee2445fdC88fA49B9d02FB8ee3633f45Bef48`

- **SOL Solana:** `HNQhe6SCoU5UDZicFKMbYjQNv9Muh39WaEWbZayQ9Nn8`

- **TRC20 USDT:** `TBJrguLia8tvydsQ2CotUDTYtCiLDA4nPW`

- **TON USDT:** `UQAdAhVxOr9LS07DDQh0vNzX2575Eu0eOByjImY1yheatXgr`

