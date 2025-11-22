export const useFormat = () => {
    const currency = (amount, currency = 'RUB') => {
        const valid = ['RUB', 'USD', 'EUR', 'GBP', 'CNY', 'JPY', 'KZT'];
        let cur = (currency || 'RUB').toUpperCase();
        if (cur === 'RUR')
            cur = 'RUB';
        if (!valid.includes(cur))
            cur = 'RUB';
        try {
            return new Intl.NumberFormat('ru-RU', {
                style: 'currency',
                currency: cur,
                minimumFractionDigits: 0,
                maximumFractionDigits: 2
            }).format(amount);
        }
        catch {
            return new Intl.NumberFormat('ru-RU', {
                minimumFractionDigits: 0,
                maximumFractionDigits: 2
            }).format(amount) + ' ' + cur;
        }
    };
    const date = (s) => {
        if (!s)
            return 'Неизвестно';
        return new Date(s).toLocaleDateString('ru-RU', {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    };
    const time = (s) => {
        if (!s)
            return 'Неизвестно';
        return new Date(s).toLocaleTimeString('ru-RU', {
            hour: '2-digit',
            minute: '2-digit'
        });
    };
    const debounce = (fn, ms) => {
        let t;
        return (...args) => {
            clearTimeout(t);
            t = setTimeout(() => fn(...args), ms);
        };
    };
    return { currency, date, time, debounce };
};
