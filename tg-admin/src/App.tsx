import React from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import { AppShell } from '@mantine/core'
import BottomNavigation from './components/BottomNavigation'
import { BroadcastsProvider } from './context/BroadcastsContext'
import { PromosProvider } from './context/PromosContext'
import { useTelegram } from './hooks/useTelegram'
import TelegramSecurityGuard from './components/Security/TelegramSecurityGuard'
import '@mantine/core/styles.css';
import './styles/global.css';

const AppContent: React.FC = () => {
  const navigate = useNavigate()
  useTelegram() // Initialize Telegram SDK

  const handleNavigation = (path: string) => {
    navigate(path)
  }

  return (
    <AppShell
      footer={{ height: 70 }}
      padding="md"
      style={{
        paddingBottom: 'var(--tg-safe-area-inset-bottom, 0px)'
      }}
    >
      <AppShell.Main style={{ paddingTop: 'calc(var(--tg-content-safe-area-inset-top, 56px) + 24px)' }}>
        <Outlet />
      </AppShell.Main>

      <AppShell.Footer>
        <BottomNavigation onSelect={handleNavigation} />
      </AppShell.Footer>
    </AppShell>
  )
}

const App: React.FC = () => {
  return (
    <TelegramSecurityGuard>
      <BroadcastsProvider>
        <PromosProvider>
          <AppContent />
        </PromosProvider>
      </BroadcastsProvider>
    </TelegramSecurityGuard>
  )
}

export default App