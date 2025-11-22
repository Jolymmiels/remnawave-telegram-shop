import React, { createContext, useContext, useState, ReactNode } from 'react'
import { http } from '@/lib/http'

interface Broadcast {
  ID: number
  Content: string
  Type: string
  Language: string
  CreatedAt: string
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
  create: (payload: { content: string; type: string; language?: string }) => Promise<void>
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
    // Prevent duplicate requests
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

  const create = async (payload: { content: string; type: string; language?: string }) => {
    const newBroadcast = await http.post('/api/broadcasts', payload)
    // Add the new broadcast to the beginning of the array (since it's sorted by created_at desc)
    setItems(prev => [newBroadcast, ...prev])
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
      create,
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