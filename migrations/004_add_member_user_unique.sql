-- Migration 004: 数据库层面强制"一人一团队"约束
-- 在 team_members.user_id 上添加唯一索引

-- 如果表中有违反约束的数据，需要先清理（本场景表已清空）
ALTER TABLE team_members ADD UNIQUE INDEX `idx_member_user_id` (`user_id`);
