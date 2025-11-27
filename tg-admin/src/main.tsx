import React from 'react'
import ReactDOM from 'react-dom/client'
import {RouterProvider} from 'react-router-dom'
import {MantineProvider} from '@mantine/core'
import {Notifications} from '@mantine/notifications'
import {router} from './router'

import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/charts/styles.css'
import "./style.css";
import {theme} from "@/theme";
import { getTelegramSafeAreaMargins } from './lib/telegram-safe-area'

const AppWrapper = () => (
  <MantineProvider defaultColorScheme="dark" theme={theme}> 
    <Notifications 
      position="bottom-right" 
      styles={{
        root: {
          ...getTelegramSafeAreaMargins(),
          bottom: 80,
          pointerEvents: 'none'
        },
        notification: {
          pointerEvents: 'auto'
        }
      }}
    />
    <RouterProvider router={router}/>
  </MantineProvider>
)

ReactDOM.createRoot(document.getElementById('root')!).render(
  import.meta.env.DEV ? (
    <React.StrictMode>
      <AppWrapper />
    </React.StrictMode>
  ) : (
    <AppWrapper />
  )
)