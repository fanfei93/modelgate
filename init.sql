-- 初始化 MySQL 数据库
-- 创建 new_api 数据库供 new-api 服务使用

CREATE DATABASE IF NOT EXISTS new_api DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON new_api.* TO 'modelgate'@'%';
FLUSH PRIVILEGES;
