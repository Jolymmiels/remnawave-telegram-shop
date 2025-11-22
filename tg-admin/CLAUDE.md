# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the admin frontend for the Remnawave Telegram Shop Bot, built as a React + TypeScript application with Mantine UI components. The admin panel is designed to be used as a Telegram Web App and provides statistics and broadcast management functionality.

### Key Architecture Components

- **Frontend (tg-admin/)**: React 18 + TypeScript SPA with Vite build system and Mantine UI
- **Backend (Go service)**: Located in parent directory, provides API endpoints at `/api/*`
- **Integration**: Communicates with Go backend via HTTP API and authenticates using Telegram WebApp initData

## Development Commands

### Frontend (tg-admin/)
```bash
npm run dev        # Start development server with hot reload
npm run build      # Build for production (TypeScript compilation + Vite build)
npm run preview    # Preview production build locally
```

### Backend (parent directory)
```bash
go run main.go     # Start Go backend server (typically runs on :8080)
docker compose up -d  # Run full stack with Docker
```

## Tech Stack

**Frontend:**
- React 18 with TypeScript
- Mantine UI component library for modern, accessible components
- React Router DOM for routing
- Custom React Context for state management
- Vite for bundling and development

**Backend Integration:**
- Communicates with Go backend via `/api/*` endpoints
- Uses Telegram WebApp initData for authentication
- Proxy configuration in vite.config.ts routes `/api` to `localhost:8080`

## Key Application Structure

### State Management (React Context)
- `UIContext.tsx` - UI state (menu open/close)
- `BroadcastsContext.tsx` - Managing broadcast messages and campaigns
- Future contexts for stats and user management

### Main Features
1. **Statistics Dashboard** - Purchase and user analytics (placeholder components)
2. **Broadcast Management** - Create and send messages to user segments (all/active/inactive users)
3. **Telegram Integration** - Native Telegram WebApp theming and authentication

### Custom Hooks
- `useTelegram()` - Integration with Telegram WebApp APIs, URL cleanup, and haptic feedback support
- `useFormat()` - Date, time, and currency formatting utilities

### API Communication
The frontend uses a custom HTTP client (`src/lib/http.ts`) that:
- Automatically includes Telegram WebApp initData in headers
- Handles JSON requests/responses
- Provides simple `http.get()` and `http.post()` methods

### Routing
Uses React Router DOM with these main routes:
- `/purchases` - Purchase statistics view
- `/users` - User statistics view  
- `/broadcasts` - Broadcast management

## Development Notes

- The app is configured to run as a Telegram WebApp and expects to be embedded within Telegram
- `useTelegram()` hook provides integration with Telegram WebApp APIs and handles URL cleanup
- Theme classes are applied automatically based on Telegram's theme
- Backend API expects Telegram authentication via the `Telegram-Init-Data` header
- Vite proxy configuration allows frontend development against local backend on port 8080
- Mantine provides comprehensive UI components with built-in accessibility and theming
- Uses hash-based routing to avoid server-side routing conflicts with Telegram WebApp parameters
- Server automatically handles Telegram WebApp URL parameters and serves the React app correctly