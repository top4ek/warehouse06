package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_resolvesStorageDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{StorageDir: dir}

	normalize(cfg)

	want, err := filepath.Abs(dir)
	require.NoError(t, err)
	assert.Equal(t, want, cfg.StorageDir)
}

func TestNormalize_emptyStorageDir(t *testing.T) {
	cfg := &Config{}
	normalize(cfg)
	assert.Empty(t, cfg.StorageDir)
}

func TestMustLoad_missingFileUsesDefaults(t *testing.T) {
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))
	t.Setenv("STORAGE_DIR", "")

	cfg := MustLoad()
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, ":memory:", cfg.DSN)
}

func TestMustLoad_noConfigPathUsesEnvAndDefaults(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("SERVER_PORT", "9090")

	cfg := MustLoad()
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "development", cfg.Env)
}

func TestMustLoad_yamlOnly(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: 8123\n"), 0o644))

	t.Setenv("CONFIG_PATH", path)
	cfg := MustLoad()
	assert.Equal(t, 8123, cfg.Port)
}

func TestMustLoad_yamlThenEnv_envWins(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: 8123\n"), 0o644))

	t.Setenv("CONFIG_PATH", path)
	t.Setenv("SERVER_PORT", "9000")
	cfg := MustLoad()
	assert.Equal(t, 9000, cfg.Port)
}

func TestValidateStorageURL(t *testing.T) {
	assert.NoError(t, validateStorageURL(""))
	assert.NoError(t, validateStorageURL("https://github.com/user/repo.git"))
	assert.NoError(t, validateStorageURL("git@github.com:user/repo.git"))
	assert.NoError(t, validateStorageURL("/srv/git/storage"))

	assert.Error(t, validateStorageURL("--upload-pack=touch /tmp/pwned"))
	assert.Error(t, validateStorageURL("-oProxyCommand=evil"))
	assert.Error(t, validateStorageURL("ext::sh -c 'touch /tmp/pwned'"))
	assert.Error(t, validateStorageURL("fd::17"))
	assert.Error(t, validateStorageURL("EXT::sh -c evil"))
}

func TestSyncInterval(t *testing.T) {
	assert.Equal(t, time.Duration(0), (&Config{Sync: Sync{IntervalHours: 0}}).SyncInterval())
	assert.Equal(t, 6*time.Hour, (&Config{Sync: Sync{IntervalHours: 6}}).SyncInterval())
}
