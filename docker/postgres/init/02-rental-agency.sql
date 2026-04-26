-- Car rental agency sample schema
-- All tables live in the "rental" schema so data persists across container restarts.

CREATE SCHEMA IF NOT EXISTS rental;
SET search_path = rental;

CREATE TABLE IF NOT EXISTS branch (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  name TEXT NOT NULL,
  city TEXT NOT NULL,
  region TEXT NOT NULL,
  branch_type TEXT NOT NULL,
  opened_on DATE NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  manager_rep_id BIGINT
);

CREATE TABLE IF NOT EXISTS rep (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  branch_id BIGINT REFERENCES branch (id),
  manager_id BIGINT REFERENCES rep (id),
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  email TEXT NOT NULL,
  role TEXT NOT NULL,
  hire_date DATE NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  quarterly_quota NUMERIC NOT NULL DEFAULT 0,
  used_car_certified BOOLEAN NOT NULL DEFAULT false
);

DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'branch_manager_rep_fk'
      AND table_schema = 'rental'
  ) THEN
    ALTER TABLE branch
      ADD CONSTRAINT branch_manager_rep_fk
      FOREIGN KEY (manager_rep_id) REFERENCES rep (id);
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS vendor (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  home_branch_id BIGINT REFERENCES branch (id),
  rating NUMERIC NOT NULL,
  preferred_vendor BOOLEAN NOT NULL DEFAULT false,
  support_email TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fleet (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  branch_id BIGINT REFERENCES branch (id),
  vendor_id BIGINT REFERENCES vendor (id),
  code TEXT NOT NULL,
  segment TEXT NOT NULL,
  active_vehicle_count INTEGER NOT NULL,
  reserve_vehicle_count INTEGER NOT NULL,
  utilization_target NUMERIC NOT NULL
);

CREATE TABLE IF NOT EXISTS vehicle_purchase (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  vendor_id BIGINT REFERENCES vendor (id),
  branch_id BIGINT REFERENCES branch (id),
  rep_id BIGINT REFERENCES rep (id),
  fleet_id BIGINT REFERENCES fleet (id),
  vin TEXT NOT NULL UNIQUE,
  purchase_date DATE NOT NULL,
  purchase_price NUMERIC NOT NULL,
  acquisition_channel TEXT NOT NULL,
  warranty_months INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS vehicle (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  fleet_id BIGINT REFERENCES fleet (id),
  vendor_id BIGINT REFERENCES vendor (id),
  purchase_id BIGINT REFERENCES vehicle_purchase (id),
  current_branch_id BIGINT REFERENCES branch (id),
  vin TEXT NOT NULL UNIQUE,
  plate_number TEXT NOT NULL UNIQUE,
  model_name TEXT NOT NULL,
  model_year INTEGER NOT NULL,
  class_code TEXT NOT NULL,
  status TEXT NOT NULL,
  odometer_km INTEGER NOT NULL,
  acquired_cost NUMERIC NOT NULL,
  estimated_resale_value NUMERIC NOT NULL
);

CREATE TABLE IF NOT EXISTS customer (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  home_branch_id BIGINT REFERENCES branch (id),
  assigned_rep_id BIGINT REFERENCES rep (id),
  referred_vendor_id BIGINT REFERENCES vendor (id),
  last_contract_id BIGINT,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  email TEXT NOT NULL,
  status TEXT NOT NULL,
  loyalty_tier TEXT NOT NULL,
  prior_contract_count INTEGER NOT NULL DEFAULT 0,
  repeat_customer BOOLEAN NOT NULL DEFAULT false,
  lifetime_rental_days INTEGER NOT NULL DEFAULT 0,
  total_spend NUMERIC NOT NULL DEFAULT 0,
  last_contract_date DATE
);

CREATE TABLE IF NOT EXISTS rental_offer (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  customer_id BIGINT REFERENCES customer (id),
  vehicle_id BIGINT REFERENCES vehicle (id),
  rep_id BIGINT REFERENCES rep (id),
  pickup_branch_id BIGINT REFERENCES branch (id),
  dropoff_branch_id BIGINT REFERENCES branch (id),
  quoted_daily_rate NUMERIC NOT NULL,
  quoted_total NUMERIC NOT NULL,
  quoted_days INTEGER NOT NULL,
  offer_status TEXT NOT NULL,
  discount_amount NUMERIC NOT NULL DEFAULT 0,
  requires_deposit BOOLEAN NOT NULL DEFAULT false,
  created_on DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS rental_contract (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  offer_id BIGINT REFERENCES rental_offer (id),
  customer_id BIGINT REFERENCES customer (id),
  vehicle_id BIGINT REFERENCES vehicle (id),
  branch_id BIGINT REFERENCES branch (id),
  rep_id BIGINT REFERENCES rep (id),
  contract_number TEXT NOT NULL UNIQUE,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  actual_total NUMERIC NOT NULL,
  actual_days INTEGER NOT NULL,
  mileage_allowance_km INTEGER NOT NULL,
  status TEXT NOT NULL,
  extension_count INTEGER NOT NULL DEFAULT 0,
  damage_fee NUMERIC NOT NULL DEFAULT 0,
  returned_late BOOLEAN NOT NULL DEFAULT false
);

DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'customer_last_contract_fk'
      AND table_schema = 'rental'
  ) THEN
    ALTER TABLE customer
      ADD CONSTRAINT customer_last_contract_fk
      FOREIGN KEY (last_contract_id) REFERENCES rental_contract (id);
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS resale_sale (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  vehicle_id BIGINT REFERENCES vehicle (id),
  customer_id BIGINT REFERENCES customer (id),
  branch_id BIGINT REFERENCES branch (id),
  rep_id BIGINT REFERENCES rep (id),
  vendor_id BIGINT REFERENCES vendor (id),
  sale_date DATE NOT NULL,
  list_price NUMERIC NOT NULL,
  sale_price NUMERIC NOT NULL,
  margin_amount NUMERIC NOT NULL,
  warranty_months INTEGER NOT NULL DEFAULT 0,
  buyer_type TEXT NOT NULL,
  trade_in BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS branch_active_idx ON branch (active);
CREATE INDEX IF NOT EXISTS rep_branch_idx ON rep (branch_id);
CREATE INDEX IF NOT EXISTS vehicle_current_branch_idx ON vehicle (current_branch_id);
CREATE INDEX IF NOT EXISTS vehicle_vendor_idx ON vehicle (vendor_id);
CREATE INDEX IF NOT EXISTS customer_assigned_rep_idx ON customer (assigned_rep_id);
CREATE INDEX IF NOT EXISTS customer_last_contract_idx ON customer (last_contract_id);
CREATE INDEX IF NOT EXISTS rental_offer_customer_idx ON rental_offer (customer_id);
CREATE INDEX IF NOT EXISTS rental_offer_vehicle_idx ON rental_offer (vehicle_id);
CREATE INDEX IF NOT EXISTS rental_contract_customer_idx ON rental_contract (customer_id);
CREATE INDEX IF NOT EXISTS rental_contract_vehicle_idx ON rental_contract (vehicle_id);
CREATE INDEX IF NOT EXISTS rental_contract_offer_idx ON rental_contract (offer_id);
CREATE INDEX IF NOT EXISTS resale_sale_vehicle_idx ON resale_sale (vehicle_id);
CREATE INDEX IF NOT EXISTS resale_sale_customer_idx ON resale_sale (customer_id);
CREATE INDEX IF NOT EXISTS resale_sale_vendor_idx ON resale_sale (vendor_id);

-- Seed data — skipped if already present
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM branch LIMIT 1) THEN RETURN; END IF;

  INSERT INTO branch (name, city, region, branch_type, opened_on, active) VALUES
    ('Ben Gurion Hub', 'Lod', 'central', 'airport', '2018-03-01', true),
    ('Jerusalem Center', 'Jerusalem', 'jerusalem', 'city', '2019-06-15', true),
    ('Haifa Port', 'Haifa', 'north', 'city', '2020-02-10', true),
    ('Eilat Sun', 'Eilat', 'south', 'tourism', '2021-11-20', true);

  INSERT INTO rep (branch_id, manager_id, first_name, last_name, email, role, hire_date, active, quarterly_quota, used_car_certified) VALUES
    (1, NULL, 'Dana', 'Levi', 'dana.levi@formql-rentals.example', 'regional-manager', '2018-03-01', true, 450000, true),
    (1, 1, 'Noam', 'Barak', 'noam.barak@formql-rentals.example', 'airport-rep', '2022-04-12', true, 220000, false),
    (2, 1, 'Maya', 'Cohen', 'maya.cohen@formql-rentals.example', 'city-rep', '2021-08-03', true, 240000, true),
    (3, 1, 'Omer', 'Adler', 'omer.adler@formql-rentals.example', 'fleet-rep', '2020-05-17', true, 260000, false),
    (4, 3, 'Lior', 'Hazan', 'lior.hazan@formql-rentals.example', 'tourism-rep', '2023-01-09', true, 180000, true);

  UPDATE branch SET manager_rep_id = 1 WHERE id IN (1, 2, 3, 4);

  INSERT INTO vendor (name, category, home_branch_id, rating, preferred_vendor, support_email) VALUES
    ('Orion Auto Group', 'manufacturer', 1, 4.7, true, 'support@orion-auto.example'),
    ('Desert Auction House', 'auction', 4, 4.1, false, 'support@desert-auction.example'),
    ('Northwind Fleet Supply', 'fleet-partner', 3, 4.5, true, 'ops@northwind.example');

  INSERT INTO fleet (branch_id, vendor_id, code, segment, active_vehicle_count, reserve_vehicle_count, utilization_target) VALUES
    (1, 1, 'AIR-ECO', 'economy', 28, 4, 0.84),
    (2, 1, 'CITY-SUV', 'suv', 16, 3, 0.78),
    (3, 3, 'PORT-VAN', 'van', 12, 2, 0.81),
    (4, 2, 'SUN-LUX', 'luxury', 7, 1, 0.72);

  INSERT INTO vehicle_purchase (vendor_id, branch_id, rep_id, fleet_id, vin, purchase_date, purchase_price, acquisition_channel, warranty_months) VALUES
    (1, 1, 2, 1, 'VIN-1001', '2023-02-14', 78000, 'factory-order', 36),
    (1, 2, 3, 2, 'VIN-2001', '2023-05-19', 132000, 'factory-order', 48),
    (3, 3, 4, 3, 'VIN-3001', '2022-12-09', 119000, 'fleet-refresh', 24),
    (2, 4, 5, 4, 'VIN-4001', '2022-07-23', 91000, 'auction-lot', 12);

  INSERT INTO vehicle (fleet_id, vendor_id, purchase_id, current_branch_id, vin, plate_number, model_name, model_year, class_code, status, odometer_km, acquired_cost, estimated_resale_value) VALUES
    (1, 1, 1, 1, 'VIN-1001', 'IL-100-11', 'Orion Swift', 2024, 'eco', 'active', 21000, 78000, 62000),
    (2, 1, 2, 2, 'VIN-2001', 'IL-200-22', 'Orion Atlas', 2024, 'suv', 'active', 18000, 132000, 109000),
    (3, 3, 3, 3, 'VIN-3001', 'IL-300-33', 'Northwind Carrier', 2023, 'van', 'service', 42000, 119000, 87000),
    (4, 2, 4, 4, 'VIN-4001', 'IL-400-44', 'Desert Crown', 2022, 'lux', 'retired', 68000, 91000, 54000);

  INSERT INTO customer (home_branch_id, assigned_rep_id, referred_vendor_id, last_contract_id, first_name, last_name, email, status, loyalty_tier, prior_contract_count, repeat_customer, lifetime_rental_days, total_spend, last_contract_date) VALUES
    (1, 2, 1, NULL, 'Yael', 'Mor', 'yael.mor@example.com', 'active', 'platinum', 8, true, 41, 28900, '2025-11-14'),
    (2, 3, 3, NULL, 'Amir', 'Segal', 'amir.segal@example.com', 'active', 'gold', 4, true, 18, 12100, '2025-10-02'),
    (3, 4, NULL, NULL, 'Ruth', 'Aviv', 'ruth.aviv@example.com', 'lead', 'standard', 0, false, 0, 0, NULL),
    (4, 5, 2, NULL, 'Daniel', 'Tal', 'daniel.tal@example.com', 'active', 'silver', 2, true, 9, 6300, '2025-09-21');

  INSERT INTO rental_offer (customer_id, vehicle_id, rep_id, pickup_branch_id, dropoff_branch_id, quoted_daily_rate, quoted_total, quoted_days, offer_status, discount_amount, requires_deposit, created_on) VALUES
    (1, 2, 2, 1, 2, 390, 1950, 5, 'accepted', 100, true, '2025-11-01'),
    (2, 1, 3, 2, 1, 210, 840, 4, 'accepted', 0, false, '2025-10-15'),
    (3, 3, 4, 3, 3, 260, 780, 3, 'quoted', 0, false, '2025-12-05'),
    (4, 4, 5, 4, 1, 540, 2160, 4, 'accepted', 180, true, '2025-09-10');

  INSERT INTO rental_contract (offer_id, customer_id, vehicle_id, branch_id, rep_id, contract_number, start_date, end_date, actual_total, actual_days, mileage_allowance_km, status, extension_count, damage_fee, returned_late) VALUES
    (1, 1, 2, 1, 2, 'CTR-24001', '2025-11-03', '2025-11-08', 2075, 5, 900, 'closed', 1, 0, false),
    (2, 2, 1, 2, 3, 'CTR-24002', '2025-10-18', '2025-10-22', 840, 4, 650, 'closed', 0, 0, false),
    (4, 4, 4, 4, 5, 'CTR-24003', '2025-09-12', '2025-09-16', 2380, 4, 700, 'closed', 0, 220, true);

  UPDATE customer SET last_contract_id = 1 WHERE id = 1;
  UPDATE customer SET last_contract_id = 2 WHERE id = 2;
  UPDATE customer SET last_contract_id = 3 WHERE id = 4;

  INSERT INTO resale_sale (vehicle_id, customer_id, branch_id, rep_id, vendor_id, sale_date, list_price, sale_price, margin_amount, warranty_months, buyer_type, trade_in) VALUES
    (4, 4, 4, 5, 2, '2026-01-11', 58500, 56400, 4700, 6, 'retail', true),
    (3, 1, 3, 4, 3, '2026-02-08', 90200, 87600, 5100, 3, 'fleet-transfer', false);
END $$;
