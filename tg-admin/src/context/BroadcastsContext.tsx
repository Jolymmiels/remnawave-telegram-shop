import React, { createContext, useContext, useState, ReactNode } from 'react'
import { http } from '@/lib/http'

interface Broadcast {
  ID: number
  Content: string
  Type: string
  Language: string
  CreatedAt: string
  Status: string
  TotalCount: number
  SentCount: number
  FailedCount: number
  BlockedCount: number
}

interface BroadcastsState {
  items: Broadcast[]
  loading: boolean
  error: string | null
  initialized: boolean
  filter: {
    type: string
    language: string
    limit: number
    offset: number
  }
  load: (reset?: boolean) => Promise<void>
  refreshActive: () => Promise<void>
  create: (payload: { content: string; type: string; language?: string }) => Promise<void>
  remove: (id: number) => Promise<void>
  setFilter: (filter: Partial<BroadcastsState['filter']>) => void
}

const BroadcastsContext = createContext<BroadcastsState | undefined>(undefined)

export const BroadcastsProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [items, setItems] = useState<Broadcast[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)
  const [filter, setFilterState] = useState({
    type: '',
    language: '',
    limit: 50,
    offset: 0
  })

  const load = async (reset = false) => {
    if (loading) return
    
    setLoading(true)
    setError(null)
    try {
      const currentFilter = reset ? { ...filter, offset: 0 } : filter
      const q = new URLSearchParams({ 
        ...currentFilter as any, 
        sort: '-created_at' 
      }).toString()
      const data = await http.get(`/api/broadcasts?${q}`)
      setItems(Array.isArray(data) ? data : [])
      setInitialized(true)
      if (reset) {
        setFilterState(prev => ({ ...prev, offset: 0 }))
      }
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  // Refresh broadcasts without showing loading state
  const refreshActive = async () => {
    try {
      const q = new URLSearchParams({ 
        ...filter as any, 
        sort: '-created_at' 
      }).toString()
      const data = await http.get(`/api/broadcasts?${q}`)
      setItems(Array.isArray(data) ? data : [])
    } catch (e) {
      // Silently fail for background refresh
    }
  }

  const create = async (payload: { content: string; type: string; language?: string }) => {
    const newBroadcast = await http.post('/api/broadcasts', payload)
    setItems(prev => [newBroadcast, ...prev])
  }

  const remove = async (id: number) => {
    await http.delete(`/api/broadcasts/${id}`)
    setItems(prev => prev.filter(item => item.ID !== id))
  }

  const setFilter = (newFilter: Partial<typeof filter>) => {
    setFilterState(prev => ({ ...prev, ...newFilter }))
  }

  return (
    <BroadcastsContext.Provider value={{
      items,
      loading,
      error,
      initialized,
      filter,
      load,
      refreshActive,
      create,
      remove,
      setFilter
    }}>
      {children}
    </BroadcastsContext.Provider>
  )
}

export const useBroadcasts = (): BroadcastsState => {
  const context = useContext(BroadcastsContext)
  if (!context) {
    throw new Error('useBroadcasts must be used within a BroadcastsProvider')
  }
  return context
}