// Development utilities for testing without Telegram

const isDevelopment = import.meta.env.DEV

// Mock Telegram init data for development testing
export const getMockTelegramInitData = (): string => {
  if (!isDevelopment) return ''
  
  // This is a mock init data string for development only
  // In production, this will never be used as it won't pass HMAC validation
  const mockData = {
    query_id: 'dev_query_id',
    user: JSON.stringify({
      id: parseInt(import.meta.env.VITE_DEV_ADMIN_ID || '123456789'),
      first_name: 'Dev',
      last_name: 'Admin',
      username: 'dev_admin',
      language_code: 'en'
    }),
    auth_date: Math.floor(Date.now() / 1000).toString(),
    hash: 'dev_mock_hash'
  }
  
  const params = new URLSearchParams(mockData)
  return params.toString()
}

// Check if we should use development bypass
export const shouldUseDevelopmentMode = (): boolean => {
  return isDevelopment && 
         import.meta.env.VITE_ENABLE_DEV_MODE === 'true' &&
         !window.location.search.includes('tgWebAppPlatform') &&
         !window.location.hash.includes('tgWebAppData')
}

// Enhanced logging for development
export const devLog = (message: string, data?: any) => {
  if (isDevelopment) {
    console.log(`[DEV] ${message}`, data || '')
  }
}

export const devWarn = (message: string, data?: any) => {
  if (isDevelopment) {
    console.warn(`[DEV] ${message}`, data || '')
  }
}

export const devError = (message: string, data?: any) => {
  if (isDevelopment) {
    console.error(`[DEV] ${message}`, data || '')
  }
}