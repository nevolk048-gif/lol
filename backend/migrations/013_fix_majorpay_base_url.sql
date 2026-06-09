-- Fix MajorPay base_url and webhook_url to use correct domain
UPDATE providers SET base_url = 'https://api.majorpay.io/api', webhook_url = 'https://api.majorpay.io/api/webhook' WHERE name = 'MajorPay';
