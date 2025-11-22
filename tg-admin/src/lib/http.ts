import { retrieveLaunchParams } from '@telegram-apps/sdk'
import { getMockTelegramInitData, shouldUseDevelopmentMode, devLog, devWarn } from './dev-utils'

// Store init data globally once extracted
let cachedInitData: string | null = null
let initDataExtracted = false

// Function to extract and cache init data early in app lifecycle
export function extractAndCacheInitData(): void {
  if (initDataExtracted) return
  
  // Try all methods immediately and cache the result
  getInitData()
}

function getInitData(): string {
  // Return cached init data if already extracted
  if (initDataExtracted && cachedInitData) {
    return cachedInitData
  }
  
  // Method 1: Try retrieveLaunchParams from @telegram-apps/sdk
  try {
    const { initDataRaw, initData } = retrieveLaunchParams()
    if (initDataRaw) {
      cachedInitData = initDataRaw as string
      initDataExtracted = true
      return cachedInitData!
    }
  } catch (error) {
    console.warn('retrieveLaunchParams failed:', error)
  }
  
  // Method 2: Try to get from Telegram WebApp object directly
  try {
    const webApp = (window as any).Telegram?.WebApp
    if (webApp?.initData) {
      cachedInitData = webApp.initData
      initDataExtracted = true
      return cachedInitData!
    }
  } catch (error) {
    console.warn('Telegram WebApp access failed:', error)
  }
  
  // Method 3: Try URL parameters directly
  try {
    const urlParams = new URLSearchParams(window.location.search)
    const tgWebAppData = urlParams.get('tgWebAppData')
    if (tgWebAppData) {
      cachedInitData = decodeURIComponent(tgWebAppData)
      initDataExtracted = true
      return cachedInitData!
    }
    
    // Also check hash parameters
    const hashParams = new URLSearchParams(window.location.hash.substring(1))
    const hashTgData = hashParams.get('tgWebAppData')
    if (hashTgData) {
      cachedInitData = decodeURIComponent(hashTgData)
      initDataExtracted = true
      return cachedInitData!
    }
  } catch (error) {
    console.warn('URL parameter fallback failed:', error)
  }
  
  // Development mode fallback
  if (shouldUseDevelopmentMode()) {
    devWarn('Using mock Telegram init data for development')
    cachedInitData = getMockTelegramInitData()
    initDataExtracted = true
    return cachedInitData!
  }
  
  // Mark as extracted even if empty to avoid repeated attempts
  initDataExtracted = true
  cachedInitData = ''
  return ''
}

async function request(method: string, url: string, body?: any) {
  const initData = getInitData()
  
  // Log warning if init data is missing for debugging
  if (!initData && url.startsWith('/api/')) {
    console.warn('Making API request without Telegram init data:', method, url)
    console.warn('Debug info:', {
      userAgent: navigator.userAgent,
      url: window.location.href,
      hasTelegramWebApp: !!(window as any).Telegram?.WebApp,
      searchParams: window.location.search,
      hashParams: window.location.hash
    })
  }
  
  const headers: Record<string, string> = { 
    'Telegram-Init-Data': initData
  }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  
  const resp = await fetch(url, { 
    method, 
    headers, 
    body: body !== undefined ? JSON.stringify(body) : undefined 
  })
  
  const text = await resp.text()
  if (!resp.ok) {
    // Enhanced error logging for authentication issues
    if (resp.status === 401 || resp.status === 403) {
      console.error('Authentication failed:', {
        status: resp.status,
        method,
        url,
        hasInitData: !!initData,
        initDataLength: initData.length,
        response: text
      })
    }
    throw new Error(`HTTP ${resp.status}: ${text || 'failure'}`)
  }
  
  try { 
    return JSON.parse(text) 
  } catch { 
    return text 
  }
}

export const http = { 
  get: (u: string) => request('GET', u), 
  post: (u: string, b: any) => request('POST', u, b),
  put: (u: string, b: any) => request('PUT', u, b),
  delete: (u: string) => request('DELETE', u)
}