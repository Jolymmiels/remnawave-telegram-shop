import { useEffect } from 'react';
import { init, retrieveLaunchParams, hapticFeedback, themeParams, miniApp, viewport } from '@telegram-apps/sdk';
import { extractAndCacheInitData } from '../lib/http';
export const useTelegram = () => {
    const applyThemeClasses = () => {
        try {
            // Mount theme params if not already mounted
            if (!themeParams.isMounted()) {
                themeParams.mountSync();
            }
            const root = document.documentElement;
            const state = themeParams.state();
            if (state.bgColor)
                root.style.setProperty('--tg-theme-bg-color', state.bgColor);
            if (state.textColor)
                root.style.setProperty('--tg-theme-text-color', state.textColor);
            if (state.hintColor)
                root.style.setProperty('--tg-theme-hint-color', state.hintColor);
            if (state.linkColor)
                root.style.setProperty('--tg-theme-link-color', state.linkColor);
            if (state.buttonColor)
                root.style.setProperty('--tg-theme-button-color', state.buttonColor);
            if (state.buttonTextColor)
                root.style.setProperty('--tg-theme-button-text-color', state.buttonTextColor);
            if (state.secondaryBgColor)
                root.style.setProperty('--tg-theme-secondary-bg-color', state.secondaryBgColor);
        }
        catch (error) {
            console.warn('Theme params not available:', error);
        }
    };
    const hapticFeedbackActions = {
        soft: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.impactOccurred('soft');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        light: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.impactOccurred('light');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        medium: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.impactOccurred('medium');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        heavy: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.impactOccurred('heavy');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        rigid: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.impactOccurred('rigid');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        selectionChanged: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.selectionChanged();
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        success: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.notificationOccurred('success');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        error: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.notificationOccurred('error');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        },
        warning: () => {
            try {
                if (hapticFeedback.isSupported()) {
                    hapticFeedback.notificationOccurred('warning');
                }
            }
            catch (error) {
                console.warn('Haptic feedback not supported:', error);
            }
        }
    };
    const getInitData = () => {
        try {
            const { initData } = retrieveLaunchParams();
            return initData || '';
        }
        catch (error) {
            console.warn('Init data not available:', error);
            return '';
        }
    };
    useEffect(() => {
        const initializeApp = async () => {
            try {
                // CRITICAL: Extract init data BEFORE any URL cleanup or redirects
                extractAndCacheInitData();
                // Initialize the SDK
                init();
                // Clean up URL if it contains Telegram WebApp data
                const currentUrl = window.location.href;
                if (currentUrl.includes('tgWebAppData')) {
                    // Extract the base URL without Telegram parameters
                    const baseUrl = currentUrl.split('tgWebAppData')[0].replace(/[?&]$/, '');
                    // Redirect to clean URL with hash routing
                    window.location.replace(baseUrl + '#/');
                    return;
                }
                // Initialize MiniApp if supported
                try {
                    if (miniApp.isSupported()) {
                        if (!miniApp.isMounted()) {
                            miniApp.mount();
                        }
                        miniApp.ready();
                    }
                }
                catch (error) {
                    console.warn('MiniApp not available:', error);
                }
                // Initialize Viewport and request fullscreen
                try {
                    // Check if viewport mount is available
                    if (viewport.mount.isAvailable()) {
                        if (!viewport.isMounted()) {
                            await viewport.mount();
                        }
                        // Bind CSS variables for safe area insets
                        if (viewport.bindCssVars.isAvailable()) {
                            viewport.bindCssVars();
                        }
                        // Expand viewport first
                        if (viewport.expand.isAvailable()) {
                            viewport.expand();
                        }
                        // Then request fullscreen mode
                        if (viewport.requestFullscreen.isAvailable()) {
                            await viewport.requestFullscreen();
                        }
                    }
                }
                catch (error) {
                    console.warn('Viewport/Fullscreen not available:', error);
                }
                applyThemeClasses();
            }
            catch (error) {
                console.warn('Telegram SDK initialization failed:', error);
            }
        };
        initializeApp();
    }, []);
    return {
        miniApp: miniApp.isSupported() ? miniApp : null,
        viewport: viewport.mount.isAvailable() ? viewport : null,
        hapticFeedback: hapticFeedbackActions,
        themeParams: themeParams.isMounted() ? themeParams : null,
        getInitData,
        applyThemeClasses
    };
};
