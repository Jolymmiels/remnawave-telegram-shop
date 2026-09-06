# Remnawave Telegram Shop

[![Stars](https://img.shields.io/github/stars/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/stargazers)
[![Forks](https://img.shields.io/github/forks/Jolymmiels/remnawave-telegram-shop.svg?style=social)](https://github.com/Jolymmiels/remnawave-telegram-shop/network/members)
[![Issues](https://img.shields.io/github/issues/Jolymmiels/remnawave-telegram-shop.svg)](https://github.com/Jolymmiels/remnawave-telegram-shop/issues)

Telegram bot for selling and managing [Remnawave](https://remna.st/) subscriptions.

> Installation, configuration, updates, and usage are covered in the [documentation](https://docs.rwp.rw/ru/).

## What the bot can do

### Subscriptions

- Sell and renew VPN subscriptions with configurable plans.
- Automatically create and update users in Remnawave.
- Give users their connection link directly in Telegram.
- Offer a configurable trial period.
- Apply separate traffic limits, reset strategies, tags, and internal or external squads to regular and trial users.
- Notify users before their subscription expires.
- Reward users through a referral program.

### Payments

- [YooKassa](https://yookassa.ru/developers/api)
- [Crypto Pay](https://help.crypt.bot/crypto-pay-api)
- Telegram Stars
- Tribute
- [Platega](https://platega.io): SBP, Russian cards, acquiring, international cards, and cryptocurrency

### Administration and interface

- Russian and English localization.
- Colored Telegram buttons with primary, success, and danger styles.
- Telegram Premium custom emoji in buttons.
- Configurable links to support, server status, feedback, channel, and terms of service.
- Synchronization of the local database with Remnawave using the `/sync` admin command.
- Configurable access restrictions for blocked and trusted Telegram users.
- Healthcheck endpoint for monitoring the bot, database, and Remnawave availability.
- Docker-based deployment.

## Menu photo and pinned start message

The bot can show the same photo on the start menu and all its inline-keyboard sections.
Put your own JPEG or PNG in `assets/menu.jpg` and set this in `.env`:

```dotenv
MENU_PHOTO_PATH=/assets/menu.jpg
```

The supplied `docker-compose.yaml` mounts `./assets:/assets:ro`. With an existing
Compose file, add that mount to the bot service. The file must be readable by the
container's UID 1000. For a non-Docker installation, use a local path such as
`MENU_PHOTO_PATH=./assets/menu.jpg`. Leave the value empty for text-only menus.

- Restart/recreate the bot after configuring or replacing the image. The first
  menu delivery uploads it to Telegram; subsequent deliveries reuse its `file_id`.
- The file ID, content hash and bot ID are stored in PostgreSQL, not in a
  short-lived memory cache. An unchanged image is reused after restart. A changed
  file is uploaded again on its next use; another bot has its own cache.
- Keep the original file and the database volume. While running, the bot retains
  the source in memory and can re-upload it if Telegram rejects the cached ID.
  After restart, a missing source can still use the persisted ID, but cannot be
  re-uploaded until the source is restored and the bot restarted. An invalid or
  missing uncached source causes an explicit startup error rather than silently
  disabling the photo.
- Telegram photo limits: JPEG/PNG up to 10 MB; width + height at most 10000 pixels;
  aspect ratio at most 20. **All menu captions, including customized translations
  and connection details, must fit Telegram's 1024-character limit after HTML
  parsing.** Text-only menus retain the ordinary text-message limits. Captions
  are not silently truncated.
- Inline navigation edits the caption and keyboard of the same photo message;
  the image and pin remain in place. Payment completion no longer deletes the
  menu. Existing text-only messages remain navigable; use `/start` for a new photo
  menu. `/connect` also uses the configured photo, but does not replace the start pin.
- Every `/start`, with or without a photo, sends and silently pins a new menu,
  then unpins only previously tracked start menus from this bot in that chat.
  Unrelated/manual pins are untouched. Pin state survives restart, and failed
  unpins are retried on the next `/start`. If sending or pinning the new menu fails,
  the old pin is not removed. Missing pin permissions do not block menu delivery
  and are logged (groups require the appropriate bot administrator rights).

Database migration `000006_add_menu_state` is applied automatically on startup.
The pin replacement lock assumes one running polling instance per bot token.

## Links

- [Documentation](https://docs.rwp.rw/ru/)
- [Changelog](CHANGELOG.md)
- [Issues](https://github.com/Jolymmiels/remnawave-telegram-shop/issues)
