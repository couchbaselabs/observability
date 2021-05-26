package manager

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/configuration"

	"github.com/stretchr/testify/assert"
)

// TestManagerKeysSetup checks that when a manger is created and runs it will create the keys needed for creating JWTs.
func TestManagerKeysSetup(t *testing.T) {
	testDir := t.TempDir()

	config := &configuration.Config{
		SQLiteKey:  "password",
		SQLiteDB:   filepath.Join(testDir, "db.sqlite"),
		MaxWorkers: 1,
	}

	manager, err := NewManager(config)
	assert.Nil(t, err, "Expected to be able to create the manager")

	go manager.Start(DefaultFrequencyConfiguration)
	time.Sleep(200 * time.Millisecond)
	manager.Stop()

	assert.Len(t, config.EncryptKey, 32, "expected 32 byte key")
	assert.Len(t, config.SignKey, 64, "expected 64 byte key")
}
