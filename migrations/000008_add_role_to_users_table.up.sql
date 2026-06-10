-- Add role column to users table with default 'user'
ALTER TABLE users
ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'user';
