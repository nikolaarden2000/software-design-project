CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TYPE user_role AS ENUM ('user', 'admin', 'superuser');
CREATE TYPE room_status AS ENUM ('draft', 'pending', 'published', 'rejected', 'archived');
CREATE TYPE booking_status AS ENUM ('booked', 'canceled');

CREATE TABLE IF NOT EXISTS companies (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'user'
);

CREATE TABLE IF NOT EXISTS locations (
  id SERIAL PRIMARY KEY,
  company_id INTEGER REFERENCES companies(id) NOT NULL,
  city TEXT NOT NULL,
  street TEXT NOT NULL,
  house_number TEXT NOT NULL,
  latitude DECIMAL(10, 8) NOT NULL,
  longitude DECIMAL(10, 8) NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Europe/Moscow'
);

CREATE INDEX idx_locations_city_company ON locations(city, company_id);

CREATE TABLE IF NOT EXISTS admin_locations (
  admin_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
  location_id INTEGER REFERENCES locations(id) ON DELETE CASCADE,
  PRIMARY KEY (admin_id, location_id)
);

CREATE INDEX idx_admin_locations_location_id ON admin_locations(location_id);

CREATE TABLE IF NOT EXISTS rooms (
  id SERIAL PRIMARY KEY,
  location_id INTEGER REFERENCES locations(id) NOT NULL,
  title TEXT NOT NULL,
  capacity SMALLINT NOT NULL,
  price INTEGER NOT NULL,
  description TEXT,
  images TEXT[] NOT NULL DEFAULT '{}',
  available_from TIME NOT NULL,
  available_to TIME NOT NULL,

  status room_status NOT NULL DEFAULT 'published',
  rejection_reason TEXT,
  created_by INTEGER REFERENCES users(id),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

  CONSTRAINT images_limit CHECK (coalesce(array_length(images, 1), 0) <= 5),
  CONSTRAINT room_capacity_positive CHECK (capacity > 0),
  CONSTRAINT room_price_positive CHECK (price > 0),
  CONSTRAINT room_available_time_valid CHECK (available_from < available_to)
);

CREATE INDEX idx_rooms_location_status ON rooms(location_id, status);
CREATE INDEX idx_rooms_status ON rooms(status);

CREATE TABLE IF NOT EXISTS bookings (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id) NOT NULL,
  room_id INTEGER REFERENCES rooms(id) NOT NULL,
  start_time TIMESTAMP WITH TIME ZONE NOT NULL,
  end_time TIMESTAMP WITH TIME ZONE NOT NULL,
  status booking_status NOT NULL DEFAULT 'booked',
  total_price INTEGER NOT NULL,
  CONSTRAINT booking_time_valid CHECK (start_time < end_time)
);

ALTER TABLE bookings
  ADD CONSTRAINT bookings_no_overlap
  EXCLUDE USING gist (
    room_id WITH =,
    tstzrange(start_time, end_time) WITH &&
  )
  WHERE (status <> 'canceled');

CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_room_id ON bookings(room_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_start_time ON bookings(start_time);