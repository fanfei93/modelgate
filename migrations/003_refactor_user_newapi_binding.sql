-- Migration 003: 重构用户与 new-api 绑定关系
-- 目标: user 与 new-api 用户一一对应，团队不再拥有 new-api 用户

-- 1. 在 users 表中添加 new-api 绑定字段
ALTER TABLE users
    ADD COLUMN new_api_user_id INT DEFAULT 0,
    ADD COLUMN new_api_password VARCHAR(256) DEFAULT '';

-- 2. 在 teams 表中移除 unique 约束并设为可空默认值
--    注意: 由于保留了字段兼容性，只移除 uniqueIndex 约束
--    如果之前有 uniqueIndex，需要根据数据库类型处理
-- MySQL:
ALTER TABLE teams
    DROP INDEX IF EXISTS `idx_teams_new_api_user_id`,
    MODIFY COLUMN new_api_user_id INT DEFAULT 0,
    MODIFY COLUMN new_api_password VARCHAR(256) DEFAULT '';

-- 3. team_members 表已有所需的字段，无需变更
