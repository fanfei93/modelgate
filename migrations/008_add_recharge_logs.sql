-- Migration 008: 充值审计日志表
CREATE TABLE IF NOT EXISTS recharge_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id BIGINT UNSIGNED NOT NULL,
    team_name VARCHAR(128) NOT NULL,
    operator_id BIGINT UNSIGNED NOT NULL,
    operator_name VARCHAR(64) NOT NULL,
    amount BIGINT NOT NULL COMMENT '充值金额（分）',
    balance_before BIGINT NOT NULL COMMENT '充值前余额（分）',
    balance_after BIGINT NOT NULL COMMENT '充值后余额（分）',
    remark VARCHAR(256) DEFAULT '' COMMENT '备注',
    ip VARCHAR(64) DEFAULT '' COMMENT '操作者 IP',
    created_at DATETIME(3) NOT NULL,
    INDEX idx_recharge_logs_team_id (team_id),
    INDEX idx_recharge_logs_operator_id (operator_id),
    INDEX idx_recharge_logs_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
