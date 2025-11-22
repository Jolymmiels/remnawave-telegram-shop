import React from 'react';
import ReactDOM from 'react-dom/client';
import { RouterProvider } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { router } from './router';
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import '@mantine/charts/styles.css';
import "./style.css";
import { theme } from "@/theme";
import { getTelegramSafeAreaMargins } from './lib/telegram-safe-area';
const AppWrapper = () => (<MantineProvider defaultColorScheme="dark" theme={theme}> 
    <Notifications position="top-right" styles={{
        root: getTelegramSafeAreaMargins()
    }}/>
    <RouterProvider router={router}/>
  </MantineProvider>);
ReactDOM.createRoot(document.getElementById('root')).render(import.meta.env.DEV ? (<React.StrictMode>
      <AppWrapper />
    </React.StrictMode>) : (<AppWrapper />));
