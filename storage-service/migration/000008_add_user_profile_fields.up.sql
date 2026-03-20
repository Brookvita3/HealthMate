ALTER TABLE users
ADD COLUMN phone TEXT,
ADD COLUMN address TEXT,
ADD COLUMN gender TEXT CHECK (gender IN ('male', 'female', 'other')),
ADD COLUMN birthday DATE,
ADD COLUMN weight NUMERIC(5, 2),
ADD COLUMN height NUMERIC(5, 2),
ADD COLUMN blood_group TEXT;
