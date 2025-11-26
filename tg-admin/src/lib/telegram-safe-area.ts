// Utility functions for Telegram WebApp safe area handling
// Based on official Telegram Mini Apps documentation:
// - SafeAreaInset: device safe area (notches, navigation bars)
// - ContentSafeAreaInset: content area (Telegram header, etc.)

export const getTelegramSafeAreaStyles = () => ({
  content: {
    paddingTop: 'var(--tg-content-safe-area-inset-top, 0px)',
    paddingRight: 'var(--tg-content-safe-area-inset-right, 0px)',
    paddingBottom: 'var(--tg-content-safe-area-inset-bottom, 0px)',
    paddingLeft: 'var(--tg-content-safe-area-inset-left, 0px)',
  }
})

export const getTelegramSafeAreaMargins = () => ({
  marginTop: 'var(--tg-safe-area-inset-top, 0px)',
  marginRight: 'var(--tg-safe-area-inset-right, 0px)',
  marginBottom: 'var(--tg-safe-area-inset-bottom, 0px)',
  marginLeft: 'var(--tg-safe-area-inset-left, 0px)',
})

export const getTelegramSafeAreaDropdownStyles = () => ({
  marginTop: 'var(--tg-content-safe-area-inset-top, 0px)',
})

// CSS custom properties for Telegram WebApp safe areas
export const TELEGRAM_SAFE_AREA_CSS_VARS = {
  // Device safe area (notches, navigation bars)
  SAFE_TOP: 'var(--tg-safe-area-inset-top, 0px)',
  SAFE_RIGHT: 'var(--tg-safe-area-inset-right, 0px)', 
  SAFE_BOTTOM: 'var(--tg-safe-area-inset-bottom, 0px)',
  SAFE_LEFT: 'var(--tg-safe-area-inset-left, 0px)',
  // Content safe area (Telegram header, etc.)
  CONTENT_TOP: 'var(--tg-content-safe-area-inset-top, 0px)',
  CONTENT_RIGHT: 'var(--tg-content-safe-area-inset-right, 0px)',
  CONTENT_BOTTOM: 'var(--tg-content-safe-area-inset-bottom, 0px)',
  CONTENT_LEFT: 'var(--tg-content-safe-area-inset-left, 0px)',
} as const