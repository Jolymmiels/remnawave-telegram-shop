import React, { createContext, useContext, useState, ReactNode, useCallback } from 'react'
import { http } from '@/lib/http'

export interface Promo {
  id: number
  code: string
  bonus_days: number
  max_uses?: number
  used_count: number
  expires_at?: string
  active: boolean
  created_at: string
}

export interface CreatePromoData {
  code: string
  bonus_days: number
  max_uses?: number
  expires_at?: string
}

interface PromosContextType {
  items: Promo[]
  loading: boolean
  error: string | null
  initialized: boolean
  load: () => Promise<void>
  create: (data: CreatePromoData) => Promise<void>
  update: (id: number, active: boolean) => Promise<void>
  delete: (id: number) => Promise<void>
}

const PromosContext = createContext<PromosContextType | undefined>(undefined)

interface PromosProviderProps {
  children: ReactNode
}

export const PromosProvider: React.FC<PromosProviderProps> = ({ children }) => {
  const [items, setItems] = useState<Promo[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [initialized, setInitialized] = useState(false)

  const load = useCallback(async () => {
    // Prevent duplicate requests
    if (loading) return
    
    setLoading(true)
    setError(null)
    try {
      const response = await http.get('/api/promos')
      setItems(response)
      setInitialized(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load promos')
    } finally {
      setLoading(false)
    }
  }, [loading])

  const create = useCallback(async (data: CreatePromoData) => {
    setError(null)
    try {
      const newPromo = await http.post('/api/promos', data)
      setItems(prev => [newPromo, ...prev])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create promo')
      throw err
    }
  }, [])

  const update = useCallback(async (id: number, active: boolean) => {
    setError(null)
    try {
      await http.put(`/api/promos/${id}`, { active })
      setItems(prev => 
        prev.map(item => 
          item.id === id ? { ...item, active } : item
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update promo')
      throw err
    }
  }, [])

  const deletePromo = useCallback(async (id: number) => {
    setError(null)
    try {
      await http.delete(`/api/promos/${id}`)
      setItems(prev => prev.filter(item => item.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete promo')
      throw err
    }
  }, [])

  const value = {
    items,
    loading,
    error,
    initialized,
    load,
    create,
    update,
    delete: deletePromo
  }

  return (
    <PromosContext.Provider value={value}>
      {children}
    </PromosContext.Provider>
  )
}

export const usePromos = (): PromosContextType => {
  const context = useContext(PromosContext)
  if (!context) {
    throw new Error('usePromos must be used within a PromosProvider')
  }
  return context
}