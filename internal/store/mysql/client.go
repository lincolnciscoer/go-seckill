package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-seckill/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const pingTimeout = 3 * time.Second

func New(cfg config.MySQLConfig) (*gorm.DB, *sql.DB, error) {
	gormDB, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("open mysql with gorm: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get sql db from gorm: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("ping mysql: %w", err)
	}

	return gormDB, sqlDB, nil
}
