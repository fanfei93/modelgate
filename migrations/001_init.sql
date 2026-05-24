-- ModelGate 数据库初始化脚本 (MySQL)
-- 注意: GORM AutoMigrate 会自动创建这些表，此文件仅作为参考

CREATE DATABASE IF NOT EXISTS modelgate CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE modelgate;

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL UNIQUE,
    email         VARCHAR(128) NOT NULL UNIQUE,
    password_hash VARCHAR(256) NOT NULL,
    display_name  VARCHAR(128),
    created_at    DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at    DATETIME(3) DEFAULT NULL,
    INDEX `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teams (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    slug            VARCHAR(64)  NOT NULL UNIQUE,
    new_api_user_id INT          NOT NULL UNIQUE,
    new_api_password VARCHAR(256) DEFAULT '',
    owner_id        BIGINT UNSIGNED NOT NULL,
    balance         BIGINT       NOT NULL DEFAULT 0,
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at      DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at      DATETIME(3) DEFAULT NULL,
    INDEX `idx_teams_owner_id` (`owner_id`),
    INDEX `idx_teams_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS team_members (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id          BIGINT UNSIGNED NOT NULL,
    user_id          BIGINT UNSIGNED NOT NULL,
    role             VARCHAR(16) NOT NULL DEFAULT 'member',
    joined_at        DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    new_api_token_id INT DEFAULT 0,
    new_api_key      VARCHAR(256) DEFAULT '',
    UNIQUE KEY `uk_team_user` (`team_id`, `user_id`),
    INDEX `idx_team_members_token_id` (`new_api_token_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS transactions (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id       BIGINT UNSIGNED NOT NULL,
    type          VARCHAR(16)  NOT NULL,
    amount        BIGINT       NOT NULL,
    balance_after BIGINT       NOT NULL,
    remark        VARCHAR(256),
    created_at    DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    INDEX `idx_transactions_team_id` (`team_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
