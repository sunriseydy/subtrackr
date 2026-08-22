package database

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateCancellationNoticeDaysDefaultsExistingRowsToZero(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE subscriptions (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subscriptions (name) VALUES (?)`, "Legacy subscription").Error)

	require.NoError(t, migrateCancellationNoticeDays(db))
	require.NoError(t, migrateCancellationNoticeDays(db), "migration must be idempotent")

	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name = 'cancellation_notice_days'").Scan(&count).Error)
	require.Equal(t, int64(1), count)

	var noticeDays int
	require.NoError(t, db.Raw("SELECT cancellation_notice_days FROM subscriptions WHERE name = ?", "Legacy subscription").Row().Scan(&noticeDays))
	require.Equal(t, 0, noticeDays, "existing subscriptions must default to no notice period")
}
