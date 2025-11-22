import React from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { AppShell } from '@mantine/core';
import AppHeader from './components/AppHeader';
import BottomNavigation from './components/BottomNavigation';
import { BroadcastsProvider } from './context/BroadcastsContext';
import { PromosProvider } from './context/PromosContext';
import { useTelegram } from './hooks/useTelegram';
import TelegramSecurityGuard from './components/Security/TelegramSecurityGuard';
import '@mantine/core/styles.css';
import './styles/global.css';
const AppContent = () => {
    const navigate = useNavigate();
    const location = useLocation();
    useTelegram(); // Initialize Telegram SDK
    const handleNavigation = (path) => {
        navigate(path);
    };
    const getPageTitle = () => {
        switch (location.pathname) {
            case '/purchases':
                return 'Статистика покупок';
            case '/users':
                return '👥 Статистика пользователей';
            case '/broadcasts':
                return 'Рассылка';
            case '/promos':
                return 'Промокоды';
            default:
                return 'Админ панель';
        }
    };
    return (<AppShell header={{
            height: {
                base: 'calc(60px + var(--tg-viewport-safe-area-inset-top, 0px))',
                sm: 60
            }
        }} footer={{ height: 70 }} padding="md" style={{
            paddingBottom: 'var(--tg-viewport-safe-area-inset-bottom, 0px)'
        }}>
      <AppShell.Header style={{ border: 'none' }}>
        <AppHeader title={getPageTitle()}/>
      </AppShell.Header>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>

      <AppShell.Footer>
        <BottomNavigation onSelect={handleNavigation}/>
      </AppShell.Footer>
    </AppShell>);
};
const App = () => {
    return (<TelegramSecurityGuard>
      <BroadcastsProvider>
        <PromosProvider>
          <AppContent />
        </PromosProvider>
      </BroadcastsProvider>
    </TelegramSecurityGuard>);
};
export default App;
