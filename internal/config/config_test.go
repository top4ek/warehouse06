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

func TestMustLoad_gopherDefaults(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")
	for _, key := range []string{"GOPHER_ENABLED", "GOPHER_LISTEN_ADDR", "GOPHER_HOST", "GOPHER_PORT"} {
		t.Setenv(key, "")
	}
	cfg := MustLoad()
	assert.True(t, cfg.Gopher.Enabled)
	assert.Equal(t, ":7070", cfg.Gopher.ListenAddr)
	assert.Equal(t, "localhost", cfg.Gopher.Host)
	assert.Equal(t, 7070, cfg.Gopher.Port)
}

func TestMustLoad_gopherYAMLAndEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("gopher:\n  enabled: true\n  listen_addr: 127.0.0.1:7071\n  host: library.example.org\n  port: 70\n"), 0o644))
	t.Setenv("CONFIG_PATH", path)
	for _, key := range []string{"GOPHER_ENABLED", "GOPHER_LISTEN_ADDR", "GOPHER_HOST", "GOPHER_PORT"} {
		t.Setenv(key, "")
	}
	cfg := MustLoad()
	assert.Equal(t, Gopher{Enabled: true, ListenAddr: "127.0.0.1:7071", Host: "library.example.org", Port: 70}, cfg.Gopher)

	t.Setenv("GOPHER_ENABLED", "false")
	t.Setenv("GOPHER_LISTEN_ADDR", ":7072")
	t.Setenv("GOPHER_HOST", "other.example.org")
	t.Setenv("GOPHER_PORT", "7073")
	cfg = MustLoad()
	assert.Equal(t, Gopher{Enabled: false, ListenAddr: ":7072", Host: "other.example.org", Port: 7073}, cfg.Gopher)
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

func TestPublicStorageURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"empty", "", ""},
		{"https", "https://github.com/top4ek/retro_files", "https://github.com/top4ek/retro_files"},
		{"git suffix stripped", "https://github.com/top4ek/retro_files.git", "https://github.com/top4ek/retro_files"},
		{"trailing slash stripped", "https://github.com/top4ek/retro_files/", "https://github.com/top4ek/retro_files"},
		{"http kept", "http://git.example.org/retro", "http://git.example.org/retro"},
		{"credentials stripped", "https://user:token@github.com/top4ek/retro_files.git", "https://github.com/top4ek/retro_files"},
		{"ssh remote", "git@github.com:top4ek/retro_files.git", ""},
		{"ssh scheme", "ssh://git@github.com/top4ek/retro_files.git", ""},
		{"local path", "/srv/git/storage", ""},
		{"file url", "file:///srv/git/storage", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{StorageURL: tt.url}
			assert.Equal(t, tt.want, cfg.PublicStorageURL())
		})
	}
}
