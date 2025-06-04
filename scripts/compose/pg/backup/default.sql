-- PostgreSQL seed data script for product and transaction tables
-- This script creates three related tables with sample data for demonstration and testing

-- Drop tables if they exist to ensure clean setup
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;

-- Create products table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    price FLOAT NOT NULL,
    stock_quantity INT NOT NULL,
    weight_kg FLOAT,
    dimensions TEXT,
    is_available BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

-- Create customers table
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT,
    address TEXT,
    city TEXT,
    country TEXT,
    postal_code TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    loyalty_points INT DEFAULT 0,
    registration_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INT NOT NULL,
    unit_price FLOAT NOT NULL,
    total_amount FLOAT NOT NULL,
    transaction_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payment_method TEXT NOT NULL,
    is_completed BOOLEAN DEFAULT TRUE,
    is_refunded BOOLEAN DEFAULT FALSE,
    shipping_address TEXT,
    tracking_number TEXT,
    notes TEXT,
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Insert sample data into products table
INSERT INTO products (id, name, description, category, price, stock_quantity, weight_kg, dimensions, is_available, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', 'Wireless Headphones', 'Noise-cancelling wireless headphones with 20-hour battery life', 'Electronics', 149.99, 45, 0.25, '7.5 x 6.5 x 3.2 inches', TRUE, '2023-01-15 10:00:00', '2023-05-20 14:30:00'),
('22222222-2222-2222-2222-222222222222', 'Smartwatch', 'Fitness tracking smartwatch with heart rate monitor', 'Electronics', 199.99, 30, 0.05, '1.5 x 1.5 x 0.4 inches', TRUE, '2023-02-10 09:15:00', '2023-06-01 11:45:00'),
('33333333-3333-3333-3333-333333333333', 'Solar Charger', 'Portable solar charger for mobile devices', 'Electronics', 59.99, 20, 0.3, '6.0 x 3.0 x 0.8 inches', TRUE, '2023-03-05 14:20:00', '2023-05-15 16:10:00'),
('44444444-4444-4444-4444-444444444444', 'Ergonomic Keyboard', 'Mechanical keyboard with customizable RGB lighting', 'Computer Accessories', 129.99, 15, 0.9, '17.3 x 5.2 x 1.4 inches', TRUE, '2023-01-20 11:30:00', '2023-04-10 09:45:00'),
('55555555-5555-5555-5555-555555555555', 'Ultrawide Monitor', '34-inch curved ultrawide monitor for gaming and productivity', 'Computer Accessories', 499.99, 8, 7.2, '32.0 x 14.5 x 9.0 inches', TRUE, '2023-02-25 13:45:00', '2023-05-05 10:30:00'),
('66666666-6666-6666-6666-666666666666', 'Bluetooth Speaker', 'Waterproof bluetooth speaker with 360-degree sound', 'Audio', 89.99, 25, 0.6, '4.0 x 4.0 x 7.0 inches', TRUE, '2023-03-10 15:20:00', '2023-06-02 12:15:00'),
('77777777-7777-7777-7777-777777777777', 'External SSD', '1TB portable solid state drive with USB-C connection', 'Storage', 159.99, 12, 0.1, '3.5 x 2.0 x 0.4 inches', TRUE, '2023-01-30 10:45:00', '2023-04-15 14:20:00'),
('88888888-8888-8888-8888-888888888888', 'Wireless Mouse', 'Ergonomic wireless mouse with adjustable DPI', 'Computer Accessories', 49.99, 40, 0.12, '4.5 x 2.8 x 1.5 inches', TRUE, '2023-02-05 09:30:00', '2023-05-10 11:00:00'),
('99999999-9999-9999-9999-999999999999', 'Laptop Backpack', 'Water-resistant backpack with anti-theft features', 'Accessories', 79.99, 18, 0.8, '18.0 x 12.0 x 6.0 inches', TRUE, '2023-03-15 12:10:00', '2023-06-05 15:30:00'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Mechanical Pencil Set', 'Professional drafting mechanical pencil set', 'Office Supplies', 24.99, 50, 0.15, '6.5 x 3.0 x 1.0 inches', TRUE, '2023-01-25 14:15:00', '2023-04-20 10:45:00'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Desk Lamp', 'LED desk lamp with adjustable brightness and color temperature', 'Home Office', 39.99, 22, 1.2, '5.0 x 5.0 x 18.0 inches', TRUE, '2023-02-15 16:30:00', '2023-05-25 13:20:00'),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Coffee Maker', 'Programmable coffee maker with thermal carafe', 'Kitchen Appliances', 119.99, 10, 3.5, '9.0 x 7.5 x 14.0 inches', TRUE, '2023-03-20 08:45:00', '2023-06-10 09:15:00'),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Yoga Mat', 'Non-slip yoga mat with carrying strap', 'Fitness', 29.99, 35, 1.0, '68.0 x 24.0 x 0.2 inches', TRUE, '2023-01-05 11:20:00', '2023-04-25 15:40:00'),
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'Air Purifier', 'HEPA air purifier for rooms up to 500 sq ft', 'Home Appliances', 179.99, 7, 5.2, '10.0 x 10.0 x 20.0 inches', TRUE, '2023-02-20 13:10:00', '2023-05-30 12:30:00'),
('ffffffff-ffff-ffff-ffff-ffffffffffff', 'Digital Drawing Tablet', 'Graphics tablet with pressure-sensitive pen', 'Art Supplies', 249.99, 9, 0.7, '12.0 x 8.0 x 0.3 inches', TRUE, '2023-03-25 10:25:00', '2023-06-15 14:50:00');

-- Insert sample data into customers table
INSERT INTO customers (id, first_name, last_name, email, phone, address, city, country, postal_code, is_active, loyalty_points, registration_date) VALUES
('aaaaaaaa-1111-1111-1111-111111111111', 'Alice', 'Johnson', 'alice.johnson@email.com', '+1-555-123-4567', '123 Maple Street', 'Portland', 'USA', '97201', TRUE, 250, '2022-06-15 09:30:00'),
('bbbbbbbb-2222-2222-2222-222222222222', 'Raj', 'Patel', 'raj.patel@email.com', '+44-20-1234-5678', '45 High Street', 'London', 'UK', 'SW1A 1AA', TRUE, 180, '2022-07-20 14:45:00'),
('cccccccc-3333-3333-3333-333333333333', 'Sophie', 'Müller', 'sophie.muller@email.com', '+49-30-1234-5678', 'Berliner Str. 42', 'Berlin', 'Germany', '10115', TRUE, 320, '2022-05-10 11:15:00'),
('dddddddd-4444-4444-4444-444444444444', 'Juan', 'Garcia', 'juan.garcia@email.com', '+34-91-123-4567', 'Calle Mayor 28', 'Madrid', 'Spain', '28013', TRUE, 150, '2022-08-05 16:20:00'),
('eeeeeeee-5555-5555-5555-555555555555', 'Yuki', 'Tanaka', 'yuki.tanaka@email.com', '+81-3-1234-5678', '1-1-1 Shibuya', 'Tokyo', 'Japan', '150-0002', TRUE, 210, '2022-09-12 08:45:00'),
('ffffffff-6666-6666-6666-666666666666', 'Maria', 'Silva', 'maria.silva@email.com', '+55-11-1234-5678', 'Av. Paulista 1000', 'São Paulo', 'Brazil', '01310-100', TRUE, 90, '2022-10-18 13:30:00'),
('77777777-7777-7777-7777-777777777777', 'Ahmed', 'Hassan', 'ahmed.hassan@email.com', '+20-2-1234-5678', '123 Tahrir Square', 'Cairo', 'Egypt', '11511', TRUE, 75, '2022-11-25 10:15:00'),
('88888888-8888-8888-8888-888888888888', 'Emma', 'Wilson', 'emma.wilson@email.com', '+61-2-1234-5678', '42 Bondi Road', 'Sydney', 'Australia', '2026', TRUE, 280, '2022-04-30 15:40:00'),
('99999999-9999-9999-9999-999999999999', 'Chen', 'Wei', 'chen.wei@email.com', '+86-10-1234-5678', '100 Nanjing Road', 'Shanghai', 'China', '200000', TRUE, 195, '2022-12-05 09:20:00'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Olivia', 'Brown', 'olivia.brown@email.com', '+1-555-987-6543', '789 Oak Avenue', 'Toronto', 'Canada', 'M5V 2T6', TRUE, 130, '2023-01-10 12:50:00'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Liam', 'Anderson', 'liam.anderson@email.com', '+1-555-456-7890', '456 Pine Road', 'Seattle', 'USA', '98101', TRUE, 220, '2023-02-15 14:10:00'),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Aisha', 'Khan', 'aisha.khan@email.com', '+92-51-1234-5678', '25 Jinnah Avenue', 'Islamabad', 'Pakistan', '44000', TRUE, 65, '2023-03-20 11:30:00'),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Lucas', 'Dubois', 'lucas.dubois@email.com', '+33-1-1234-5678', '15 Rue de Rivoli', 'Paris', 'France', '75001', TRUE, 170, '2022-08-25 16:45:00'),
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'Isabella', 'Rossi', 'isabella.rossi@email.com', '+39-06-1234-5678', 'Via del Corso 12', 'Rome', 'Italy', '00186', TRUE, 240, '2022-07-15 10:20:00'),
('ffffffff-ffff-ffff-ffff-ffffffffffff', 'Noah', 'Kim', 'noah.kim@email.com', '+82-2-1234-5678', '123 Gangnam-daero', 'Seoul', 'South Korea', '06000', TRUE, 110, '2023-01-05 13:15:00');

-- Insert sample data into transactions table
INSERT INTO transactions (id, customer_id, product_id, quantity, unit_price, total_amount, transaction_date, payment_method, is_completed, is_refunded, shipping_address, tracking_number, notes) VALUES
('11111111-aaaa-1111-aaaa-111111111111', 'aaaaaaaa-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 1, 149.99, 149.99, '2023-04-10 14:30:00', 'Credit Card', TRUE, FALSE, '123 Maple Street, Portland, USA', 'TRK123456789', 'Standard delivery'),
('22222222-bbbb-2222-bbbb-222222222222', 'bbbbbbbb-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 1, 199.99, 199.99, '2023-04-12 10:15:00', 'PayPal', TRUE, FALSE, '45 High Street, London, UK', 'TRK234567890', 'Express delivery'),
('33333333-cccc-3333-cccc-333333333333', 'cccccccc-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 2, 59.99, 119.98, '2023-04-15 16:45:00', 'Credit Card', TRUE, FALSE, 'Berliner Str. 42, Berlin, Germany', 'TRK345678901', 'Gift wrapped'),
('44444444-dddd-4444-dddd-444444444444', 'dddddddd-4444-4444-4444-444444444444', '44444444-4444-4444-4444-444444444444', 1, 129.99, 129.99, '2023-04-18 09:20:00', 'Debit Card', TRUE, FALSE, 'Calle Mayor 28, Madrid, Spain', 'TRK456789012', NULL),
('55555555-eeee-5555-eeee-555555555555', 'eeeeeeee-5555-5555-5555-555555555555', '55555555-5555-5555-5555-555555555555', 1, 499.99, 499.99, '2023-04-20 13:10:00', 'Credit Card', TRUE, FALSE, '1-1-1 Shibuya, Tokyo, Japan', 'TRK567890123', 'Signature required'),
('66666666-ffff-6666-ffff-666666666666', 'ffffffff-6666-6666-6666-666666666666', '66666666-6666-6666-6666-666666666666', 2, 89.99, 179.98, '2023-04-22 15:30:00', 'PayPal', TRUE, FALSE, 'Av. Paulista 1000, São Paulo, Brazil', 'TRK678901234', NULL),
('77777777-aaaa-7777-aaaa-777777777777', '77777777-7777-7777-7777-777777777777', '77777777-7777-7777-7777-777777777777', 1, 159.99, 159.99, '2023-04-25 11:45:00', 'Credit Card', TRUE, FALSE, '123 Tahrir Square, Cairo, Egypt', 'TRK789012345', 'Standard delivery'),
('88888888-aaaa-8888-aaaa-888888888888', '88888888-8888-8888-8888-888888888888', '88888888-8888-8888-8888-888888888888', 3, 49.99, 149.97, '2023-04-28 14:20:00', 'Debit Card', TRUE, FALSE, '42 Bondi Road, Sydney, Australia', 'TRK890123456', NULL),
('99999999-aaaa-9999-aaaa-999999999999', '99999999-9999-9999-9999-999999999999', '99999999-9999-9999-9999-999999999999', 1, 79.99, 79.99, '2023-05-01 10:05:00', 'Credit Card', TRUE, FALSE, '100 Nanjing Road, Shanghai, China', 'TRK901234567', 'Express delivery'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 2, 24.99, 49.98, '2023-05-03 16:15:00', 'PayPal', TRUE, FALSE, '789 Oak Avenue, Toronto, Canada', 'TRK012345678', 'Gift wrapped'),
('bbbbbbbb-aaaa-bbbb-aaaa-bbbbbbbbbbbb', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 1, 39.99, 39.99, '2023-05-05 09:30:00', 'Credit Card', TRUE, FALSE, '456 Pine Road, Seattle, USA', 'TRK123456780', NULL),
('cccccccc-aaaa-cccc-aaaa-cccccccccccc', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 1, 119.99, 119.99, '2023-05-08 13:40:00', 'Debit Card', TRUE, FALSE, '25 Jinnah Avenue, Islamabad, Pakistan', 'TRK234567801', 'Standard delivery'),
('dddddddd-aaaa-dddd-aaaa-dddddddddddd', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 2, 29.99, 59.98, '2023-05-10 15:25:00', 'Credit Card', TRUE, FALSE, '15 Rue de Rivoli, Paris, France', 'TRK345678012', NULL),
('eeeeeeee-aaaa-eeee-aaaa-eeeeeeeeeeee', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 1, 179.99, 179.99, '2023-05-12 11:10:00', 'PayPal', TRUE, FALSE, 'Via del Corso 12, Rome, Italy', 'TRK456780123', 'Signature required'),
('ffffffff-aaaa-ffff-aaaa-ffffffffffff', 'ffffffff-ffff-ffff-ffff-ffffffffffff', 'ffffffff-ffff-ffff-ffff-ffffffffffff', 1, 249.99, 249.99, '2023-05-15 14:50:00', 'Credit Card', TRUE, FALSE, '123 Gangnam-daero, Seoul, South Korea', 'TRK567801234', 'Express delivery'),
('55555555-aaaa-5555-aaaa-555555555555', 'aaaaaaaa-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 1, 199.99, 199.99, '2023-05-18 10:30:00', 'Credit Card', TRUE, FALSE, '123 Maple Street, Portland, USA', 'TRK678012345', NULL),
('22222222-aaaa-2222-aaaa-222222222222', 'bbbbbbbb-2222-2222-2222-222222222222', '33333333-3333-3333-3333-333333333333', 1, 59.99, 59.99, '2023-05-20 16:05:00', 'PayPal', TRUE, FALSE, '45 High Street, London, UK', 'TRK789123456', 'Standard delivery'),
('33333333-aaaa-3333-aaaa-333333333333', 'cccccccc-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 1, 129.99, 129.99, '2023-05-22 13:15:00', 'Debit Card', TRUE, TRUE, 'Berliner Str. 42, Berlin, Germany', 'TRK890234567', 'Refunded due to defect'),
('44444444-aaaa-4444-aaaa-444444444444', 'dddddddd-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555', 1, 499.99, 499.99, '2023-05-25 09:45:00', 'Credit Card', TRUE, FALSE, 'Calle Mayor 28, Madrid, Spain', 'TRK901345678', 'Signature required');
