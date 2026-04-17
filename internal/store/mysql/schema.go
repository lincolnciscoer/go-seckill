package mysql

import (
	"database/sql"
	"fmt"
)

func EnsureSchema(db *sql.DB) error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS mq_consume_logs (
		  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  message_id VARCHAR(128) NOT NULL,
		  business_key VARCHAR(128) NOT NULL,
		  status TINYINT NOT NULL DEFAULT 1,
		  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
		  UNIQUE KEY uk_mq_consume_logs_message_id (message_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
		`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}

	return nil
}
