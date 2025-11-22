import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'node:url';
import webfontDownload from 'vite-plugin-webfont-dl';
export default defineConfig({
    plugins: [react(), webfontDownload()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
    },
    server: {
        allowedHosts: ['d31f02474e1e.ngrok-free.app'],
        host: true,
        proxy: {
            '/api': {
                target: 'http://localhost:8080',
                changeOrigin: true,
                secure: false,
            },
        },
    },
});
