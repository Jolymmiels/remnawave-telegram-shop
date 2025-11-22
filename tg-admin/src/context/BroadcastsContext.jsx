import React, { createContext, useContext, useState } from 'react';
import { http } from '@/lib/http';
const BroadcastsContext = createContext(undefined);
export const BroadcastsProvider = ({ children }) => {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const [initialized, setInitialized] = useState(false);
    const [filter, setFilterState] = useState({
        type: '',
        language: '',
        limit: 50,
        offset: 0
    });
    const load = async (reset = false) => {
        // Prevent duplicate requests
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            const currentFilter = reset ? { ...filter, offset: 0 } : filter;
            const q = new URLSearchParams({
                ...currentFilter,
                sort: '-created_at'
            }).toString();
            const data = await http.get(`/api/broadcasts?${q}`);
            setItems(Array.isArray(data) ? data : []);
            setInitialized(true);
            if (reset) {
                setFilterState(prev => ({ ...prev, offset: 0 }));
            }
        }
        catch (e) {
            setError(e.message);
        }
        finally {
            setLoading(false);
        }
    };
    const create = async (payload) => {
        const newBroadcast = await http.post('/api/broadcasts', payload);
        // Add the new broadcast to the beginning of the array (since it's sorted by created_at desc)
        setItems(prev => [newBroadcast, ...prev]);
    };
    const setFilter = (newFilter) => {
        setFilterState(prev => ({ ...prev, ...newFilter }));
    };
    return (<BroadcastsContext.Provider value={{
            items,
            loading,
            error,
            initialized,
            filter,
            load,
            create,
            setFilter
        }}>
      {children}
    </BroadcastsContext.Provider>);
};
export const useBroadcasts = () => {
    const context = useContext(BroadcastsContext);
    if (!context) {
        throw new Error('useBroadcasts must be used within a BroadcastsProvider');
    }
    return context;
};
