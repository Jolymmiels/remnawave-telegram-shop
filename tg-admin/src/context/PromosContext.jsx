import React, { createContext, useContext, useState, useCallback } from 'react';
import { http } from '@/lib/http';
const PromosContext = createContext(undefined);
export const PromosProvider = ({ children }) => {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const [initialized, setInitialized] = useState(false);
    const load = useCallback(async () => {
        // Prevent duplicate requests
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            const response = await http.get('/api/promos');
            setItems(response);
            setInitialized(true);
        }
        catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load promos');
        }
        finally {
            setLoading(false);
        }
    }, [loading]);
    const create = useCallback(async (data) => {
        setError(null);
        try {
            const newPromo = await http.post('/api/promos', data);
            setItems(prev => [newPromo, ...prev]);
        }
        catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to create promo');
            throw err;
        }
    }, []);
    const update = useCallback(async (id, active) => {
        setError(null);
        try {
            await http.put(`/api/promos/${id}`, { active });
            setItems(prev => prev.map(item => item.id === id ? { ...item, active } : item));
        }
        catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update promo');
            throw err;
        }
    }, []);
    const deletePromo = useCallback(async (id) => {
        setError(null);
        try {
            await http.delete(`/api/promos/${id}`);
            setItems(prev => prev.filter(item => item.id !== id));
        }
        catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete promo');
            throw err;
        }
    }, []);
    const value = {
        items,
        loading,
        error,
        initialized,
        load,
        create,
        update,
        delete: deletePromo
    };
    return (<PromosContext.Provider value={value}>
      {children}
    </PromosContext.Provider>);
};
export const usePromos = () => {
    const context = useContext(PromosContext);
    if (!context) {
        throw new Error('usePromos must be used within a PromosProvider');
    }
    return context;
};
