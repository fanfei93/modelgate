-- 站点配置表
CREATE TABLE IF NOT EXISTS site_settings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `key` VARCHAR(64) NOT NULL UNIQUE,
    `value` VARCHAR(2048) NOT NULL DEFAULT '',
    comment VARCHAR(256) DEFAULT '',
    updated_at DATETIME(3) NOT NULL,
    INDEX idx_site_settings_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入默认配置
INSERT INTO site_settings (`key`, `value`, comment, updated_at) VALUES
('site_name', 'ModelGate', '站点名称', NOW()),
('menu_arena_visible', 'true', '是否显示操练场菜单', NOW()),
('menu_docs_visible', 'true', '是否显示文档菜单', NOW()),
('menu_docs_url', '/docs', '文档菜单跳转链接', NOW());
