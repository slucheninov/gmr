package version

import (
	"regexp"
	"testing"
)

func TestVersionFormat(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	// Allow plain semver (0.6.0) or build-time injected vX.Y.Z(-suffix).
	re := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[\w.]+)?$|^dev-[a-f0-9]+$`)
	if !re.MatchString(Version) {
		t.Errorf("Version %q does not match expected pattern", Version)
	}
}
