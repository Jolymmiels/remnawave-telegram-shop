import React, { Suspense } from 'react'
import { createHashRouter, Navigate } from 'react-router-dom'
import { Loader, Center } from '@mantine/core'
import App from './App'

// Dynamic imports for code splitting
const PurchasesView = React.lazy(() => import('./components/Stats/PurchasesView'))
const UsersView = React.lazy(() => import('./components/Stats/UsersView'))
const BroadcastsView = React.lazy(() => import('./components/Broadcasts/List'))
const PromosView = React.lazy(() => import('./components/Promos/List'))
const UserManagement = React.lazy(() => import('./components/Users/UserManagement'))
const UserDetailsPage = React.lazy(() => import('./components/Users/UserDetailsPage'))
const SettingsPage = React.lazy(() => import('./components/Settings/SettingsPage'))

// Loading component
const PageLoader = () => (
  <Center h={200}>
    <Loader size="md" />
  </Center>
)

export const router = createHashRouter([
  {
    path: '/',
    element: <App />,
    children: [
      { path: '/', element: <Navigate to="/purchases" replace /> },
      { 
        path: '/purchases', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <PurchasesView />
          </Suspense>
        ),
        handle: { title: 'Статистика покупок' }
      },
      { 
        path: '/users', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <UsersView />
          </Suspense>
        ),
        handle: { title: 'Статистика пользователей' }
      },
      { 
        path: '/broadcasts', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <BroadcastsView />
          </Suspense>
        ),
        handle: { title: 'Рассылка' }
      },
      { 
        path: '/promos', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <PromosView />
          </Suspense>
        ),
        handle: { title: 'Промокоды' }
      },
      { 
        path: '/user-management', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <UserManagement />
          </Suspense>
        ),
        handle: { title: 'Управление пользователями' }
      },
      { 
        path: '/user/:telegramId', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <UserDetailsPage />
          </Suspense>
        ),
        handle: { title: 'Детали пользователя' }
      },
      { 
        path: '/settings', 
        element: (
          <Suspense fallback={<PageLoader />}>
            <SettingsPage />
          </Suspense>
        ),
        handle: { title: 'Настройки' }
      },
      // Catch-all route for any unmatched paths (including Telegram WebApp data)
      { 
        path: '*', 
        element: <Navigate to="/purchases" replace />
      }
    ]
  }
])