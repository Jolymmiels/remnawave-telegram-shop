// Utility functions for Telegram WebApp safe area handling

export const getTelegramSafeAreaStyles = () => ({
  content: {
    margin: 'var(--tg-viewport-safe-area-inset-top, 0px) var(--tg-viewport-safe-area-inset-right, 0px) var(--tg-viewport-safe-area-inset-bottom, 0px) var(--tg-viewport-safe-area-inset-left, 0px)',
  }
})

export const getTelegramSafeAreaMargins = () => ({
  marginTop: 'calc(var(--tg-viewport-safe-area-inset-top, 0px) + 16px)',
  marginRight: 'calc(var(--tg-viewport-safe-area-inset-right, 0px) + 16px)',
  marginBottom: 'calc(var(--tg-viewport-safe-area-inset-bottom, 0px) + 16px)',
  marginLeft: 'calc(var(--tg-viewport-safe-area-inset-left, 0px) + 16px)',
})

export const getTelegramSafeAreaDropdownStyles = () => ({
  margin: 'var(--tg-viewport-safe-area-inset-top, 0px) var(--tg-viewport-safe-area-inset-right, 0px) var(--tg-viewport-safe-area-inset-bottom, 0px) var(--tg-viewport-safe-area-inset-left, 0px)',
})

// CSS custom properties for Telegram WebApp safe areas
export const TELEGRAM_SAFE_AREA_CSS_VARS = {
  TOP: 'var(--tg-viewport-safe-area-inset-top, 0px)',
  RIGHT: 'var(--tg-viewport-safe-area-inset-right, 0px)', 
  BOTTOM: 'var(--tg-viewport-safe-area-inset-bottom, 0px)',
  LEFT: 'var(--tg-viewport-safe-area-inset-left, 0px)',
} as const