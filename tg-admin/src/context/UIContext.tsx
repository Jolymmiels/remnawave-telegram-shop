import React, { createContext, useContext, useState, ReactNode } from 'react'

interface UIState {
  menuOpen: boolean
  toggleMenu: () => void
  closeMenu: () => void
}

const UIContext = createContext<UIState | undefined>(undefined)

export const UIProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [menuOpen, setMenuOpen] = useState(false)

  const toggleMenu = () => setMenuOpen(prev => !prev)
  const closeMenu = () => setMenuOpen(false)

  return (
    <UIContext.Provider value={{ menuOpen, toggleMenu, closeMenu }}>
      {children}
    </UIContext.Provider>
  )
}

export const useUI = (): UIState => {
  const context = useContext(UIContext)
  if (!context) {
    throw new Error('useUI must be used within a UIProvider')
  }
  return context
}