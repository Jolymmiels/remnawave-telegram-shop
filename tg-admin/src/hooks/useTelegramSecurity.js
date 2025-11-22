import { useState, useEffect } from 'react';
import { retrieveLaunchParams } from '@telegram-apps/sdk';
export const useTelegramSecurity = () => {
    const [state, setState] = useState({
        isInTelegram: false,
        hasInitData: false,
        isValidEnvironment: false,
        isLoading: true,
        error: null,
        telegramId: null,
    });
    useEffect(() => {
        const checkTelegramEnvironment = () => {
            try {
                // Check if we're in Telegram WebApp environment
                const isInTelegram = !!(window.Telegram?.WebApp ||
                    window.location.search.includes('tgWebAppPlatform') ||
                    window.location.hash.includes('tgWebAppData'));
                // Try to get launch parameters
                let hasInitData = false;
                let telegramId = null;
                try {
                    const { initData, initDataRaw } = retrieveLaunchParams();
                    hasInitData = !!(initData && initDataRaw);
                    // Extract user ID from initData if available
                    if (initData && typeof initData === 'object' && 'user' in initData) {
                        const userData = initData.user;
                        if (userData && userData.id) {
                            telegramId = userData.id;
                        }
                    }
                }
                catch (error) {
                    console.warn('Could not retrieve Telegram launch params:', error);
                    hasInitData = false;
                }
                // Additional checks for Telegram WebApp specific features
                const hasTelegramWebApp = !!(window.Telegram?.WebApp);
                const hasUserAgent = /Telegram/i.test(navigator.userAgent);
                const isValidEnvironment = isInTelegram && (hasInitData || hasTelegramWebApp || hasUserAgent);
                setState({
                    isInTelegram,
                    hasInitData,
                    isValidEnvironment,
                    isLoading: false,
                    error: isValidEnvironment ? null : 'Application must be opened within Telegram',
                    telegramId,
                });
            }
            catch (error) {
                console.error('Telegram security check failed:', error);
                setState({
                    isInTelegram: false,
                    hasInitData: false,
                    isValidEnvironment: false,
                    isLoading: false,
                    error: error instanceof Error ? error.message : 'Security check failed',
                    telegramId: null,
                });
            }
        };
        // Check immediately
        checkTelegramEnvironment();
        // Also check after a short delay in case Telegram SDK needs time to initialize
        const timeoutId = setTimeout(checkTelegramEnvironment, 1000);
        return () => clearTimeout(timeoutId);
    }, []);
    return state;
};
// Helper function to check if user is admin (to be used with backend verification)
export const checkAdminStatus = async (telegramId) => {
    try {
        const response = await fetch('/api/auth/check-admin', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Telegram-Init-Data': getInitData(),
            },
            body: JSON.stringify({ telegram_id: telegramId })
        });
        if (!response.ok) {
            return false;
        }
        const result = await response.json();
        return result.is_admin === true;
    }
    catch (error) {
        console.error('Admin status check failed:', error);
        return false;
    }
};
function getInitData() {
    try {
        const { initDataRaw } = retrieveLaunchParams();
        return initDataRaw || '';
    }
    catch (error) {
        console.warn('Init data not available:', error);
        return '';
    }
}
