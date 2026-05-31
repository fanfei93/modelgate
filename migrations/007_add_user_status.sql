-- Migration 007: 用户表添加 status 字段（active / disabled）
-- 绑定校验失败时将用户标记为 disabled，阻止登录

ALTER TABLE users ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active' AFTER new_api_password;
