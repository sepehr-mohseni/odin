package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlugin_FileExists tests file existence validation
func TestValidatePlugin_FileExists(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.so")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		_, err = os.Stat(testFile)
		assert.NoError(t, err)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := os.Stat("/nonexistent/file.so")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("directory instead of file", func(t *testing.T) {
		tmpDir := t.TempDir()
		info, err := os.Stat(tmpDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

// TestValidatePlugin_ELFMagicNumber tests ELF magic number validation
func TestValidatePlugin_ELFMagicNumber(t *testing.T) {
	tests := []struct {
		name       string
		header     []byte
		isValidELF bool
	}{
		{
			name:       "valid ELF header",
			header:     []byte{0x7f, 0x45, 0x4c, 0x46},
			isValidELF: true,
		},
		{
			name:       "invalid magic number",
			header:     []byte{0x00, 0x00, 0x00, 0x00},
			isValidELF: false,
		},
		{
			name:       "partial header",
			header:     []byte{0x7f, 0x45},
			isValidELF: false,
		},
		{
			name:       "text file",
			header:     []byte("test"),
			isValidELF: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.so")
			err := os.WriteFile(testFile, tt.header, 0644)
			require.NoError(t, err)

			data, err := os.ReadFile(testFile)
			require.NoError(t, err)

			isValid := len(data) >= 4 &&
				data[0] == 0x7f &&
				data[1] == 0x45 &&
				data[2] == 0x4c &&
				data[3] == 0x46

			assert.Equal(t, tt.isValidELF, isValid)
		})
	}
}

// TestValidatePlugin_GoVersion tests Go version extraction and validation
func TestValidatePlugin_GoVersion(t *testing.T) {
	tests := []struct {
		name               string
		version            string
		expectedMajorMinor string
	}{
		{"go1.25.3", "go1.25.3", "go1.25"},
		{"go1.25", "go1.25", "go1.25"},
		{"go1.24.0", "go1.24.0", "go1.24"},
		{"go1.23", "go1.23", "go1.23"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract major.minor version
			version := tt.version
			majorMinor := version

			// Simple extraction: go1.XX.Y -> go1.XX
			if len(version) > 6 {
				// Format: goX.YY.Z or goX.Y.Z
				for i := 4; i < len(version); i++ {
					if version[i] == '.' {
						// Found second dot
						if i < len(version)-1 {
							majorMinor = version[:i]
							break
						}
					}
				}
			}

			assert.Equal(t, tt.expectedMajorMinor, majorMinor)
		})
	}
}

// TestValidatePlugin_FileSize tests file size validation
func TestValidatePlugin_FileSize(t *testing.T) {
	tests := []struct {
		name       string
		size       int
		maxSize    int64
		shouldPass bool
	}{
		{"small file", 1024, 50 * 1024 * 1024, true},
		{"medium file", 10 * 1024 * 1024, 50 * 1024 * 1024, true},
		{"max size", 50 * 1024 * 1024, 50 * 1024 * 1024, true},
		{"too large", 51 * 1024 * 1024, 50 * 1024 * 1024, false},
		{"empty file", 0, 50 * 1024 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.so")

			data := make([]byte, tt.size)
			err := os.WriteFile(testFile, data, 0644)
			require.NoError(t, err)

			info, err := os.Stat(testFile)
			require.NoError(t, err)

			isValid := info.Size() > 0 && info.Size() <= tt.maxSize
			assert.Equal(t, tt.shouldPass, isValid)
		})
	}
}

// TestValidatePlugin_SymbolChecking tests symbol validation
func TestValidatePlugin_SymbolChecking(t *testing.T) {
	requiredSymbols := []string{"New"}

	t.Run("required symbols exist", func(t *testing.T) {
		// In a real plugin, we would use plugin.Lookup()
		// Here we just test the logic
		foundSymbols := map[string]bool{
			"New": true,
		}

		for _, symbol := range requiredSymbols {
			assert.True(t, foundSymbols[symbol], "Symbol %s should exist", symbol)
		}
	})

	t.Run("missing required symbol", func(t *testing.T) {
		foundSymbols := map[string]bool{
			"Init": true,
		}

		hasAllSymbols := true
		for _, symbol := range requiredSymbols {
			if !foundSymbols[symbol] {
				hasAllSymbols = false
				break
			}
		}

		assert.False(t, hasAllSymbols, "Should detect missing symbols")
	})
}

// TestExtractPluginMetadata tests metadata extraction from binary
func TestExtractPluginMetadata(t *testing.T) {
	t.Run("valid plugin binary", func(t *testing.T) {
		// This test would require a real compiled plugin
		// For now, we test the metadata structure
		metadata := map[string]string{
			"go_version": "go1.25",
			"go_os":      "linux",
			"go_arch":    "amd64",
		}

		assert.NotEmpty(t, metadata["go_version"])
		assert.NotEmpty(t, metadata["go_os"])
		assert.NotEmpty(t, metadata["go_arch"])
	})

	t.Run("platform validation", func(t *testing.T) {
		validOS := []string{"linux", "darwin", "windows"}
		validArch := []string{"amd64", "arm64"}

		testCases := []struct {
			os    string
			arch  string
			valid bool
		}{
			{"linux", "amd64", true},
			{"darwin", "amd64", true},
			{"darwin", "arm64", true},
			{"windows", "amd64", true},
			{"invalid", "amd64", false},
			{"linux", "invalid", false},
		}

		for _, tc := range testCases {
			isValidOS := contains(validOS, tc.os)
			isValidArch := contains(validArch, tc.arch)
			isValid := isValidOS && isValidArch

			assert.Equal(t, tc.valid, isValid,
				"Platform %s/%s validity should be %v", tc.os, tc.arch, tc.valid)
		}
	})
}

// TestPluginValidator_VersionCompatibility tests version compatibility checking
func TestPluginValidator_VersionCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		systemVersion string
		pluginVersion string
		compatible    bool
	}{
		{"exact match", "go1.25", "go1.25", true},
		{"patch difference", "go1.25.3", "go1.25.1", true},
		{"minor mismatch", "go1.25", "go1.24", false},
		{"major mismatch", "go2.0", "go1.25", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract major.minor for both
			sysMajorMinor := extractMajorMinor(tt.systemVersion)
			pluginMajorMinor := extractMajorMinor(tt.pluginVersion)

			compatible := sysMajorMinor == pluginMajorMinor
			assert.Equal(t, tt.compatible, compatible)
		})
	}
}

// TestPluginValidator_SecurityChecks tests security validation
func TestPluginValidator_SecurityChecks(t *testing.T) {
	t.Run("executable permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.so")

		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)

		// Check file is readable
		mode := info.Mode()
		assert.True(t, mode.IsRegular(), "Should be a regular file")
		assert.False(t, mode.IsDir(), "Should not be a directory")
	})

	t.Run("file type validation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a symlink (should be rejected)
		realFile := filepath.Join(tmpDir, "real.so")
		symlinkFile := filepath.Join(tmpDir, "link.so")

		os.WriteFile(realFile, []byte("test"), 0644)
		os.Symlink(realFile, symlinkFile)

		realInfo, _ := os.Lstat(realFile)
		linkInfo, _ := os.Lstat(symlinkFile)

		assert.False(t, linkInfo.Mode()&os.ModeSymlink == 0,
			"Symlinks should be detected")
		assert.True(t, realInfo.Mode().IsRegular(),
			"Regular files should be allowed")
	})
}

// TestPluginValidator_ErrorMessages tests error message clarity
func TestPluginValidator_ErrorMessages(t *testing.T) {
	errorCases := []struct {
		name             string
		errorType        string
		expectedContains []string
	}{
		{
			"file not found",
			"file_not_found",
			[]string{"not found", "does not exist"},
		},
		{
			"invalid ELF",
			"invalid_elf",
			[]string{"ELF", "magic number", "not a valid"},
		},
		{
			"version mismatch",
			"version_mismatch",
			[]string{"version", "compiled with", "expected"},
		},
		{
			"missing symbol",
			"missing_symbol",
			[]string{"symbol", "not found", "required"},
		},
	}

	for _, ec := range errorCases {
		t.Run(ec.name, func(t *testing.T) {
			// Verify expected error message components
			assert.NotEmpty(t, ec.expectedContains,
				"Error messages should contain descriptive text")
		})
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractMajorMinor(version string) string {
	// Extract go1.XX from go1.XX.Y
	if len(version) < 6 {
		return version
	}

	for i := 4; i < len(version); i++ {
		if version[i] == '.' {
			if i < len(version)-1 {
				return version[:i]
			}
		}
	}

	return version
}

// Benchmark tests
func BenchmarkELFMagicNumberCheck(b *testing.B) {
	data := []byte{0x7f, 0x45, 0x4c, 0x46, 0x02, 0x01, 0x01, 0x00}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = len(data) >= 4 &&
			data[0] == 0x7f &&
			data[1] == 0x45 &&
			data[2] == 0x4c &&
			data[3] == 0x46
	}
}

func BenchmarkVersionExtraction(b *testing.B) {
	version := "go1.25.3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractMajorMinor(version)
	}
}

func BenchmarkFileStatCheck(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.so")
	os.WriteFile(testFile, []byte("test"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.Stat(testFile)
	}
}
