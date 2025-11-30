-- Fix typo: yookasa -> yookassa in invoice_type
UPDATE purchase SET invoice_type = 'yookassa' WHERE invoice_type = 'yookasa';
