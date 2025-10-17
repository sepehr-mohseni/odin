package plugins

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"plugin"
	"runtime"
	"strings"
)

// PluginValidator provides validation for uploaded plugins
type PluginValidator struct {
	requiredGoVersion string
	requiredSymbols   []string
}

// NewPluginValidator creates a new plugin validator
func NewPluginValidator() *PluginValidator {
	return &PluginValidator{
		requiredGoVersion: runtime.Version(), // Current Go version
		requiredSymbols:   []string{"New"},   // Required exported symbols
	}
}

// ValidatePlugin performs comprehensive plugin validation
func (pv *PluginValidator) ValidatePlugin(pluginPath, expectedName string) error {
	// 1. Check file exists and is not empty
	if err := pv.validateFileExists(pluginPath); err != nil {
		return err
	}

	// 2. Validate Go version compatibility
	if err := pv.validateGoVersion(pluginPath); err != nil {
		return fmt.Errorf("Go version mismatch: %w", err)
	}

	// 3. Validate plugin can be loaded (test load)
	if err := pv.validatePluginLoadable(pluginPath); err != nil {
		return fmt.Errorf("plugin cannot be loaded: %w", err)
	}

	// 4. Validate required symbols exist
	if err := pv.validateSymbols(pluginPath); err != nil {
		return fmt.Errorf("missing required symbols: %w", err)
	}

	// 5. Basic security checks
	if err := pv.performSecurityCheck(pluginPath); err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	return nil
}

// validateFileExists checks if file exists and is not empty
func (pv *PluginValidator) validateFileExists(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot access plugin file: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("plugin file is empty")
	}

	return nil
}

// validateGoVersion checks if plugin was built with compatible Go version
func (pv *PluginValidator) validateGoVersion(pluginPath string) error {
	// Try to read build info from the binary
	info, err := buildinfo.ReadFile(pluginPath)
	if err != nil {
		// If we can't read build info, log warning but don't fail
		// Some older binaries might not have build info embedded
		return nil
	}

	pluginGoVersion := info.GoVersion
	currentGoVersion := runtime.Version()

	// For Go plugins, the version must match exactly (major.minor)
	// Extract major.minor from both versions
	pluginVersion := extractMajorMinor(pluginGoVersion)
	currentVersion := extractMajorMinor(currentGoVersion)

	if pluginVersion != currentVersion {
		return fmt.Errorf(
			"plugin built with Go %s, but gateway is running Go %s (must match exactly for plugins)",
			pluginGoVersion,
			currentGoVersion,
		)
	}

	return nil
}

// extractMajorMinor extracts major.minor version from Go version string
// e.g., "go1.25.3" -> "go1.25"
func extractMajorMinor(version string) string {
	// Remove "go" prefix if present
	version = strings.TrimPrefix(version, "go")

	// Split by dots
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return "go" + parts[0] + "." + parts[1]
	}
	return version
}

// validatePluginLoadable attempts to load the plugin to ensure it's valid
func (pv *PluginValidator) validatePluginLoadable(pluginPath string) error {
	// Try to open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// If we can open it, it's likely valid
	_ = p
	return nil
}

// validateSymbols checks if required symbols are present
func (pv *PluginValidator) validateSymbols(pluginPath string) error {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("cannot open plugin: %w", err)
	}

	// Check for required symbols
	missingSymbols := []string{}
	for _, symbolName := range pv.requiredSymbols {
		_, err := p.Lookup(symbolName)
		if err != nil {
			missingSymbols = append(missingSymbols, symbolName)
		}
	}

	if len(missingSymbols) > 0 {
		return fmt.Errorf("missing required symbols: %s", strings.Join(missingSymbols, ", "))
	}

	// Validate the New function has correct signature
	newSymbol, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("missing 'New' function")
	}

	// Try to assert to the expected function signature
	_, ok := newSymbol.(func(map[string]interface{}) (Middleware, error))
	if !ok {
		return fmt.Errorf("'New' function has incorrect signature, expected: func(map[string]interface{}) (Middleware, error)")
	}

	return nil
}

// performSecurityCheck performs basic security validation
func (pv *PluginValidator) performSecurityCheck(pluginPath string) error {
	// Read file for basic security checks
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("cannot read plugin file: %w", err)
	}

	// Check file is not too small (likely corrupted)
	if len(data) < 1024 {
		return fmt.Errorf("plugin file too small, may be corrupted")
	}

	// Check for ELF magic number (Linux shared object)
	if len(data) >= 4 {
		elfMagic := []byte{0x7f, 'E', 'L', 'F'}
		if !bytesEqual(data[:4], elfMagic) {
			return fmt.Errorf("not a valid ELF shared object file")
		}
	}

	// TODO: More sophisticated checks:
	// - Check for suspicious imports
	// - Validate symbols don't contain dangerous patterns
	// - Run static analysis tools
	// - Check digital signature (future enhancement)

	return nil
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ExtractPluginMetadata extracts metadata from plugin binary
func ExtractPluginMetadata(pluginPath string) (goVersion, goOS, goArch string, err error) {
	info, err := buildinfo.ReadFile(pluginPath)
	if err != nil {
		// Return defaults if we can't read build info
		return runtime.Version(), runtime.GOOS, runtime.GOARCH, nil
	}

	goVersion = info.GoVersion

	// Extract GOOS and GOARCH from build settings
	for _, setting := range info.Settings {
		switch setting.Key {
		case "GOOS":
			goOS = setting.Value
		case "GOARCH":
			goArch = setting.Value
		}
	}

	// Use runtime values as fallback
	if goOS == "" {
		goOS = runtime.GOOS
	}
	if goArch == "" {
		goArch = runtime.GOARCH
	}

	return goVersion, goOS, goArch, nil
}
