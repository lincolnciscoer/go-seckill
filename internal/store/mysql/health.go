package mysql

import (
	"context"
	"database/sql"
)

type healthChecker struct {
	db *sql.DB
}

func NewHealthChecker(db *sql.DB) *healthChecker {
	return &healthChecker{db: db}
}

func (c *healthChecker) Name() string {
	return "mysql"
}

func (c *healthChecker) Check(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
