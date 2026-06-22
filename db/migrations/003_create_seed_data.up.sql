-- 1. Insert Countries and capture IDs
WITH countries AS (
    INSERT INTO regions (name_fa, name_en, display_order) VALUES
    ('ایران', 'Iran', 1),
    ('ترکیه', 'Turkey', 2),
    ('امارات متحده عربی', 'United Arab Emirates', 3)
    RETURNING id, name_en
),
-- 2. Insert Expanded Iranian Provinces
iran_provinces AS (
    INSERT INTO regions (name_fa, name_en, parent_id, display_order)
    SELECT name_fa, name_en, (SELECT id FROM countries WHERE name_en = 'Iran'), display_order
    FROM (VALUES 
        ('تهران', 'Tehran', 1),
        ('اصفهان', 'Isfahan', 2),
        ('خراسان رضوی', 'Razavi Khorasan', 3),
        ('فارس', 'Fars', 4),
        ('آذربایجان شرقی', 'East Azerbaijan', 5),
        ('مازندران', 'Mazandaran', 6),
        ('گیلان', 'Gilan', 7),
        ('خوزستان', 'Khuzestan', 8),
        ('کرمان', 'Kerman', 9),
        ('البرز', 'Alborz', 10),
        ('یزد', 'Yazd', 11),
        ('هرمزگان', 'Hormozgan', 12),
        ('سیستان و بلوچستان', 'Sistan and Baluchestan', 13),
        ('کردستان', 'Kurdistan', 14),
        ('کرمانشاه', 'Kermanshah', 15)
    ) AS v(name_fa, name_en, display_order)
    RETURNING id, name_en
)
-- 3. Insert Major Cities for each Province
INSERT INTO regions (name_fa, name_en, parent_id, display_order)
SELECT name_fa, name_en, parent_id, display_order
FROM (
    -- Tehran Cities
    SELECT 'تهران', 'Tehran City', id, 1 FROM iran_provinces WHERE name_en = 'Tehran' UNION ALL
    SELECT 'شهریار', 'Shahriar', id, 2 FROM iran_provinces WHERE name_en = 'Tehran' UNION ALL
    SELECT 'ری', 'Rey', id, 3 FROM iran_provinces WHERE name_en = 'Tehran' UNION ALL
    
    -- Alborz Cities
    SELECT 'کرج', 'Karaj', id, 1 FROM iran_provinces WHERE name_en = 'Alborz' UNION ALL
    SELECT 'فردیس', 'Fardis', id, 2 FROM iran_provinces WHERE name_en = 'Alborz' UNION ALL
    
    -- Isfahan Cities
    SELECT 'اصفهان', 'Isfahan City', id, 1 FROM iran_provinces WHERE name_en = 'Isfahan' UNION ALL
    SELECT 'کاشان', 'Kashan', id, 2 FROM iran_provinces WHERE name_en = 'Isfahan' UNION ALL
    SELECT 'نجف‌آباد', 'Najafabad', id, 3 FROM iran_provinces WHERE name_en = 'Isfahan' UNION ALL
    
    -- Khuzestan Cities
    SELECT 'اهواز', 'Ahvaz', id, 1 FROM iran_provinces WHERE name_en = 'Khuzestan' UNION ALL
    SELECT 'آبادان', 'Abadan', id, 2 FROM iran_provinces WHERE name_en = 'Khuzestan' UNION ALL
    SELECT 'دزفول', 'Dezful', id, 3 FROM iran_provinces WHERE name_en = 'Khuzestan' UNION ALL
    
    -- Fars Cities
    SELECT 'شیراز', 'Shiraz', id, 1 FROM iran_provinces WHERE name_en = 'Fars' UNION ALL
    SELECT 'مرودشت', 'Marvdasht', id, 2 FROM iran_provinces WHERE name_en = 'Fars' UNION ALL
    
    -- Razavi Khorasan Cities
    SELECT 'مشهد', 'Mashhad', id, 1 FROM iran_provinces WHERE name_en = 'Razavi Khorasan' UNION ALL
    SELECT 'نیشابور', 'Neyshabur', id, 2 FROM iran_provinces WHERE name_en = 'Razavi Khorasan' UNION ALL
    
    -- East Azerbaijan Cities
    SELECT 'تبریز', 'Tabriz', id, 1 FROM iran_provinces WHERE name_en = 'East Azerbaijan' UNION ALL
    SELECT 'مراغه', 'Maragheh', id, 2 FROM iran_provinces WHERE name_en = 'East Azerbaijan' UNION ALL
    
    -- Kerman Cities
    SELECT 'کرمان', 'Kerman City', id, 1 FROM iran_provinces WHERE name_en = 'Kerman' UNION ALL
    SELECT 'سیرجان', 'Sirjan', id, 2 FROM iran_provinces WHERE name_en = 'Kerman' UNION ALL
    
    -- Hormozgan Cities
    SELECT 'بندرعباس', 'Bandar Abbas', id, 1 FROM iran_provinces WHERE name_en = 'Hormozgan' UNION ALL
    SELECT 'کیش', 'Kish', id, 2 FROM iran_provinces WHERE name_en = 'Hormozgan' UNION ALL
    SELECT 'قشم', 'Qeshm', id, 3 FROM iran_provinces WHERE name_en = 'Hormozgan' UNION ALL
    
    -- Yazd Cities
    SELECT 'یزد', 'Yazd City', id, 1 FROM iran_provinces WHERE name_en = 'Yazd' UNION ALL
    SELECT 'میبد', 'Meybod', id, 2 FROM iran_provinces WHERE name_en = 'Yazd' UNION ALL
    
    -- Sistan & Baluchestan Cities
    SELECT 'زاهدان', 'Zahedan', id, 1 FROM iran_provinces WHERE name_en = 'Sistan and Baluchestan' UNION ALL
    SELECT 'چابهار', 'Chabahar', id, 2 FROM iran_provinces WHERE name_en = 'Sistan and Baluchestan' UNION ALL
    
    -- Kurdistan & Kermanshah
    SELECT 'سنندج', 'Sanandaj', id, 1 FROM iran_provinces WHERE name_en = 'Kurdistan' UNION ALL
    SELECT 'کرمانشاه', 'Kermanshah City', id, 1 FROM iran_provinces WHERE name_en = 'Kermanshah'
) AS cities(name_fa, name_en, parent_id, display_order);

-- User seeding remains the same as they don't have parent dependencies
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


