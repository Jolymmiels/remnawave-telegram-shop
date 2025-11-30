-- Revert: yookassa -> yookasa in invoice_type
UPDATE purchase SET invoice_type = 'yookasa' WHERE invoice_type = 'yookassa';
