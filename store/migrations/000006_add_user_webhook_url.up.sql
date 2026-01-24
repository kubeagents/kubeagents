ALTER TABLE users
ADD COLUMN IF NOT EXISTS notification_webhook_url TEXT;
