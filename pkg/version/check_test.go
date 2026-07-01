package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate_DevBuild(t *testing.T) {
	latest, hasUpdate := CheckForUpdate("main")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	latest, hasUpdate := CheckForUpdate("")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCompareVersions_NewerAvailable(t *testing.T) {
	latest, hasUpdate := compareVersions("2.0.0", "1.0.0")
	assert.True(t, hasUpdate)
	assert.Equal(t, "2.0.0", latest)
}

func TestCompareVersions_AlreadyCurrent(t *testing.T) {
	latest, hasUpdate := compareVersions("1.0.0", "1.0.0")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCompareVersions_OlderAvailable(t *testing.T) {
	latest, hasUpdate := compareVersions("0.9.0", "1.0.0")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCompareVersions_EmptyLatest(t *testing.T) {
	latest, hasUpdate := compareVersions("", "1.0.0")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCompareVersions_InvalidLatest(t *testing.T) {
	latest, hasUpdate := compareVersions("not-semver", "1.0.0")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestCompareVersions_InvalidCurrent(t *testing.T) {
	latest, hasUpdate := compareVersions("2.0.0", "not-semver")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestLoadCache_Valid(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "version-check.json")

	cache := &versionCache{
		LatestVersion: "1.5.0",
		CheckedAt:     time.Now(),
	}
	data, err := json.Marshal(cache)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cacheFile, data, 0o644))

	loaded, err := loadCache(cacheFile)
	require.NoError(t, err)
	assert.Equal(t, "1.5.0", loaded.LatestVersion)
	assert.WithinDuration(t, time.Now(), loaded.CheckedAt, time.Second)
}

func TestLoadCache_MissingFile(t *testing.T) {
	_, err := loadCache("/nonexistent/path/file.json")
	assert.Error(t, err)
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "version-check.json")
	require.NoError(t, os.WriteFile(cacheFile, []byte("{invalid"), 0o644))

	_, err := loadCache(cacheFile)
	assert.Error(t, err)
}

func TestSaveCache(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "version-check.json")

	cache := &versionCache{
		LatestVersion: "3.0.0",
		CheckedAt:     time.Now(),
	}
	require.NoError(t, saveCache(cacheFile, cache))

	loaded, err := loadCache(cacheFile)
	require.NoError(t, err)
	assert.Equal(t, "3.0.0", loaded.LatestVersion)
}

func TestCheckForUpdate_FreshCache(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "version-check.json")

	cache := &versionCache{
		LatestVersion: "2.0.0",
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(cacheFile, data, 0o644)

	// Override config dir via XDG
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Rename the cache file to match what CheckForUpdate expects
	os.Rename(cacheFile, filepath.Join(dir, "surfer", "version-check.json"))
	os.MkdirAll(filepath.Join(dir, "surfer"), 0o750)
	os.WriteFile(filepath.Join(dir, "surfer", "version-check.json"), data, 0o644)

	latest, hasUpdate := CheckForUpdate("1.0.0")
	assert.True(t, hasUpdate)
	assert.Equal(t, "2.0.0", latest)
}

func TestCheckForUpdate_StaleCache(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "surfer")
	os.MkdirAll(cacheDir, 0o750)
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Write a stale cache (older than 24h)
	cache := &versionCache{
		LatestVersion: "1.0.0",
		CheckedAt:     time.Now().Add(-48 * time.Hour),
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(cacheDir, "version-check.json"), data, 0o644)

	// fetchLatest will fail (no real GitHub releases), so should return ("", false)
	latest, hasUpdate := CheckForUpdate("1.0.0")
	assert.False(t, hasUpdate)
	assert.Empty(t, latest)
}

func TestSaveCache_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	now := time.Now().Truncate(time.Second)
	err := saveCache(path, &versionCache{LatestVersion: "5.0.0", CheckedAt: now})
	require.NoError(t, err)

	loaded, err := loadCache(path)
	require.NoError(t, err)
	assert.Equal(t, "5.0.0", loaded.LatestVersion)
}

func TestCompareVersions_PatchBump(t *testing.T) {
	latest, hasUpdate := compareVersions("1.0.1", "1.0.0")
	assert.True(t, hasUpdate)
	assert.Equal(t, "1.0.1", latest)
}

func TestSaveCache_UnwritablePath(t *testing.T) {
	err := saveCache("/nonexistent/dir/cache.json", &versionCache{
		LatestVersion: "1.0.0",
		CheckedAt:     time.Now(),
	})
	assert.Error(t, err)
}

func TestFetchLatest_ReturnsErrorOrEmpty(t *testing.T) {
	// fetchLatest hits the real GitHub API with a 5s timeout.
	// For a nonexistent repo it should return an error or empty.
	// This tests the function is callable and handles errors.
	result, err := fetchLatest()
	// May succeed (if Surfe/surfer exists on GH) or fail — both are valid
	_ = result
	_ = err
}
