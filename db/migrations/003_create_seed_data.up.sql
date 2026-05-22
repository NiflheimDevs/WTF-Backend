-- Seed data for regions table (30 records)

-- Level 1: Countries
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('ایران', 'Iran', NULL, TRUE, 1),
('ترکیه', 'Turkey', NULL, TRUE, 2),
('امارات متحده عربی', 'United Arab Emirates', NULL, TRUE, 3);

-- Level 2: Iranian Provinces
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('تهران', 'Tehran', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 1),
('اصفهان', 'Isfahan', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 2),
('خراسان رضوی', 'Razavi Khorasan', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 3),
('فارس', 'Fars', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 4),
('آذربایجان شرقی', 'East Azerbaijan', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 5),
('مازندران', 'Mazandaran', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 6),
('گیلان', 'Gilan', 'f5dab899-e8cf-4c34-b4b6-5faf4cdc49a6', TRUE, 7);

-- Level 3: Tehran Province Cities
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('تهران', 'Tehran City', '5b7eae1e-5636-4096-94d9-d4316aeae5fa', TRUE, 1),
('کرج', 'Karaj', '5b7eae1e-5636-4096-94d9-d4316aeae5fa', TRUE, 2),
('شهریار', 'Shahriar', '5b7eae1e-5636-4096-94d9-d4316aeae5fa', TRUE, 3),
('ری', 'Rey', '5b7eae1e-5636-4096-94d9-d4316aeae5fa', TRUE, 4),
('پردیس', 'Pardis', '5b7eae1e-5636-4096-94d9-d4316aeae5fa', TRUE, 5);

-- Level 3: Isfahan Province Cities
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('اصفهان', 'Isfahan City', '654143ab-9360-4013-bbdf-b9bfdb3a8024', TRUE, 1),
('کاشان', 'Kashan', '654143ab-9360-4013-bbdf-b9bfdb3a8024', TRUE, 2),
('نجف‌آباد', 'Najafabad', '654143ab-9360-4013-bbdf-b9bfdb3a8024', TRUE, 3),
('خمینی‌شهر', 'Khomeinishahr', '654143ab-9360-4013-bbdf-b9bfdb3a8024', TRUE, 4);

-- Level 3: Razavi Khorasan Province Cities
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('مشهد', 'Mashhad', '713f373d-f907-48aa-9367-d4ce78549c5c', TRUE, 1),
('نیشابور', 'Neyshabur', '713f373d-f907-48aa-9367-d4ce78549c5c', TRUE, 2),
('سبزوار', 'Sabzevar', '713f373d-f907-48aa-9367-d4ce78549c5c', TRUE, 3);

-- Level 3: Fars Province Cities
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('شیراز', 'Shiraz', '264cfb48-dddc-43f5-88aa-eac9fdcc6491', TRUE, 1),
('مرودشت', 'Marvdasht', '264cfb48-dddc-43f5-88aa-eac9fdcc6491', TRUE, 2),
('جهرم', 'Jahrom', '264cfb48-dddc-43f5-88aa-eac9fdcc6491', TRUE, 3);

-- Level 3: East Azerbaijan Province Cities
INSERT INTO regions (name_fa, name_en, parent_id, is_active, display_order) VALUES
('تبریز', 'Tabriz', 'a6c0c261-f159-4091-ba67-32e131f2076e', TRUE, 1),
('مراغه', 'Maragheh', 'a6c0c261-f159-4091-ba67-32e131f2076e', TRUE, 2),
('مرند', 'Marand', 'a6c0c261-f159-4091-ba67-32e131f2076e', TRUE, 3);


INSERT INTO users (email, password_hash, full_name, role, is_active) VALUES
('admin@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'مدیر سیستم', 'admin', TRUE),
('dispatcher1@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'علی احمدی', 'dispatcher', TRUE),
('dispatcher2@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'سارا محمدی', 'dispatcher', TRUE),
('admin2@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'رضا کریمی', 'admin', TRUE),
('dispatcher3@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'فاطمه حسینی', 'dispatcher', TRUE),
('dispatcher4@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'محمد رضایی', 'dispatcher', TRUE),
('admin3@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'زهرا نوری', 'admin', TRUE),
('dispatcher5@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'حسین مرادی', 'dispatcher', TRUE),
('dispatcher6@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'مریم اکبری', 'dispatcher', TRUE),
('dispatcher7@example.com', '$2a$10$TN/7aZ4t4K3recjg9Pv5z.9t3WSQGORRiec4j1avbB/x1U6ISU8t2', 'امیر صادقی', 'dispatcher', TRUE);


