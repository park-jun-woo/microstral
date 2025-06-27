package model

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
)

type TestModel struct {
	Postgres *sql.DB
	Redis    *redis.Client
}

func NewTestModel(postgres *sql.DB, redis *redis.Client) *TestModel {
	return &TestModel{
		Postgres: postgres,
		Redis:    redis,
	}
}

func (tm *TestModel) GetTest() (string, error) {
	return "This is test.", nil
}
