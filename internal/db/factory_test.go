package db

import (
	"fmt"
	"testing"

	"github.com/nemirlev/zenexport/internal/config"
	"github.com/nemirlev/zenexport/internal/db/clickhouse"
	"github.com/nemirlev/zenexport/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewDataStore_ClickHouse(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "clickhouse",
	}

	log := logger.New()
	ds, err := NewDataStore(cfg, log)

	assert.NoError(t, err)
	assert.NotNil(t, ds)

	_, ok := ds.(*clickhouse.Store)
	assert.True(t, ok, "expected ClickHouse store")
}

func TestNewDataStore_Unsupported(t *testing.T) {
	cfg := &config.Config{
		DatabaseType: "unsupported",
	}

	log := logger.New()
	ds, err := NewDataStore(cfg, log)

	assert.Error(t, err)
	assert.Nil(t, ds)
	assert.Equal(t, fmt.Sprintf("unsupported database type: %s", cfg.DatabaseType), err.Error())
}
