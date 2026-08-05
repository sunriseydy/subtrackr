package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateAutopayPreservesExistingRowsAsUnknown(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE subscriptions (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO subscriptions (name) VALUES (?)`, "Legacy subscription").Error)

	require.NoError(t, migrateAutopay(db))
	require.NoError(t, migrateAutopay(db), "migration must be idempotent")

	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name = 'autopay'").Scan(&count).Error)
	require.Equal(t, int64(1), count)

	var autopay sql.NullBool
	require.NoError(t, db.Raw("SELECT autopay FROM subscriptions WHERE name = ?", "Legacy subscription").Row().Scan(&autopay))
	require.False(t, autopay.Valid, "existing subscriptions must remain unknown")
}
