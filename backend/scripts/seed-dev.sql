-- Development/demo data only. Never execute this file in production.
INSERT IGNORE INTO users (openid, nickname, city, ski_type, ski_level, bio) VALUES
('dev-seed-1', '雪友阿峰', '北京', 'snowboard', 'intermediate', '周末崇礼刷道'),
('dev-seed-2', '小鹿', '北京', 'ski', 'primary', '双板初级'),
('dev-seed-3', '大力', '上海', 'snowboard', 'advanced', '刻滑练习'),
('dev-seed-4', '可可', '杭州', 'both', 'intermediate', '单双板都滑'),
('dev-seed-5', '北北', '张家口', 'ski', 'advanced', '安全同行');

INSERT INTO ski_events (title, creator_id, resort_id, resort_name, event_date, start_time, depart_city, depart_area, meet_place, traffic_type, max_members, current_members, ski_type_req, level_req, purpose_tags, allow_beginner, allow_car_pool, remark, status)
SELECT CONCAT('开发示例雪局-', n), (SELECT id FROM users WHERE openid='dev-seed-1'),
       CASE WHEN MOD(n,2)=0 THEN 2 ELSE 1 END,
       CASE WHEN MOD(n,2)=0 THEN '南山滑雪场' ELSE '崇礼 · 万龙滑雪场' END,
       DATE_ADD(CURDATE(), INTERVAL n DAY), DATE_ADD(DATE_ADD(CURDATE(), INTERVAL n DAY), INTERVAL 7 HOUR),
       '北京', '朝阳', '地铁站停车场', 'self_drive', 4, 1, 'both', 'intermediate', JSON_ARRAY('刷道','互拍'), 1, 1,
       '开发环境示例数据，仅用于联调。', 'recruiting'
FROM (SELECT 1 n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9 UNION ALL SELECT 10) numbers
WHERE NOT EXISTS (SELECT 1 FROM ski_events WHERE title=CONCAT('开发示例雪局-', n));

INSERT IGNORE INTO event_members (event_id, user_id, role, status)
SELECT e.id, e.creator_id, 'creator', 'active' FROM ski_events e WHERE e.title LIKE '开发示例雪局-%';
