import React, { createContext, useContext, useState } from 'react';
const UIContext = createContext(undefined);
export const UIProvider = ({ children }) => {
    const [menuOpen, setMenuOpen] = useState(false);
    const toggleMenu = () => setMenuOpen(prev => !prev);
    const closeMenu = () => setMenuOpen(false);
    return (<UIContext.Provider value={{ menuOpen, toggleMenu, closeMenu }}>
      {children}
    </UIContext.Provider>);
};
export const useUI = () => {
    const context = useContext(UIContext);
    if (!context) {
        throw new Error('useUI must be used within a UIProvider');
    }
    return context;
};
