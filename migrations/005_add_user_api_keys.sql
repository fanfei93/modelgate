-- 用户个人 API Key 表（一个用户可拥有多个 Key）
CREATE TABLE IF NOT EXISTS `user_api_keys` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` INT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `token_id` INT NOT NULL DEFAULT 0,
  `key` VARCHAR(256) NOT NULL,
  `status` INT NOT NULL DEFAULT 1 COMMENT '1: 启用, 2: 禁用',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_userapikey_user_id` (`user_id`),
  INDEX `idx_user_api_keys_token_id` (`token_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
