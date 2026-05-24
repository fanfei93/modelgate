-- ModelGate 迁移: 添加 Team 密码字段，重构 TeamMember token 字段
-- 方案A: 成员 API Key 改为在团队 new-api 用户下创建独立 token

ALTER TABLE teams ADD COLUMN new_api_password VARCHAR(256) DEFAULT '' AFTER new_api_user_id;
-- 旧架构的 teams.new_api_key 列不再使用，先设默认值（后续可 DROP）
ALTER TABLE teams MODIFY COLUMN new_api_key VARCHAR(256) DEFAULT '';

ALTER TABLE team_members CHANGE COLUMN new_api_user_id new_api_token_id INT DEFAULT 0;
ALTER TABLE team_members ADD COLUMN new_api_key VARCHAR(256) DEFAULT '' AFTER new_api_token_id;

-- 注: GORM AutoMigrate 也会自动处理这些字段变更
