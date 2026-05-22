package semver

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		version string
		wantErr bool
	}{
		{"1.2.3", false},
		{"v1.2.3", false},
		{"0.0.1", false},
		{"1.2.3-alpha", false},
		{"1.2.3+build", false},
		{"1.2.3-alpha+build", false},
		{"", true},
		{"invalid", true},
		{"abc", true},
	}
	for _, tc := range tests {
		v, err := Parse(tc.version)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Parse(%q) expected error, got %v", tc.version, v)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tc.version, err)
			continue
		}
		if v == nil {
			t.Errorf("Parse(%q) returned nil", tc.version)
		}
	}
}

func TestMustParse(t *testing.T) {
	v := MustParse("1.2.3")
	if v == nil {
		t.Fatal("MustParse returned nil")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse should panic on invalid version")
		}
	}()
	MustParse("invalid")
}

func TestVersionString(t *testing.T) {
	v := MustParse("v1.2.3")
	if s := v.String(); s != "v1.2.3" {
		t.Errorf("Version.String() = %q, want %q", s, "v1.2.3")
	}
}

func TestVersionMajorMinorPatch(t *testing.T) {
	v := MustParse("1.2.3")
	if m := v.Major(); m != 1 {
		t.Errorf("Version.Major() = %d, want 1", m)
	}
	if m := v.Minor(); m != 2 {
		t.Errorf("Version.Minor() = %d, want 2", m)
	}
	if p := v.Patch(); p != 3 {
		t.Errorf("Version.Patch() = %d, want 3", p)
	}
}

func TestVersionPrereleaseAndMetadata(t *testing.T) {
	v := MustParse("1.2.3-alpha.1+build.42")
	if p := v.Prerelease(); p != "alpha.1" {
		t.Errorf("Version.Prerelease() = %q, want %q", p, "alpha.1")
	}
	if m := v.Metadata(); m != "build.42" {
		t.Errorf("Version.Metadata() = %q, want %q", m, "build.42")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
	}
	for _, tc := range tests {
		got, err := Compare(tc.v1, tc.v2)
		if err != nil {
			t.Errorf("Compare(%q, %q) unexpected error: %v", tc.v1, tc.v2, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.v1, tc.v2, got, tc.want)
		}
	}
}

func TestCompareError(t *testing.T) {
	_, err := Compare("invalid", "1.0.0")
	if err == nil {
		t.Error("Compare with invalid version should return error")
	}

	_, err = Compare("1.0.0", "invalid")
	if err == nil {
		t.Error("Compare with invalid version should return error")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", true},
	}
	for _, tc := range tests {
		got, err := IsNewer(tc.current, tc.latest)
		if err != nil {
			t.Errorf("IsNewer(%q, %q) unexpected error: %v", tc.current, tc.latest, err)
			continue
		}
		if got != tc.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestIsNewerError(t *testing.T) {
	_, err := IsNewer("invalid", "1.0.0")
	if err == nil {
		t.Error("IsNewer with invalid version should return error")
	}
}

func TestSatisfies(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.2.3", ">=1.0.0", true},
		{"1.2.3", ">=2.0.0", false},
		{"1.2.3", "~1.2", true},
		{"1.2.3", "~1.3", false},
		{"1.2.3", "1.x", true},
	}
	for _, tc := range tests {
		got, err := Satisfies(tc.version, tc.constraint)
		if err != nil {
			t.Errorf("Satisfies(%q, %q) unexpected error: %v", tc.version, tc.constraint, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Satisfies(%q, %q) = %v, want %v", tc.version, tc.constraint, got, tc.want)
		}
	}
}

func TestSatisfiesError(t *testing.T) {
	_, err := Satisfies("invalid", ">=1.0.0")
	if err == nil {
		t.Error("Satisfies with invalid version should return error")
	}

	_, err = Satisfies("1.0.0", "invalid constraint")
	if err == nil {
		t.Error("Satisfies with invalid constraint should return error")
	}
}

func TestMajorMinusVersion(t *testing.T) {
	v := MustParse("2.0.0")
	if v.Major() != 2 {
		t.Errorf("expected major 2, got %d", v.Major())
	}
}

func TestZeroVersionMajor(t *testing.T) {
	v := MustParse("0.0.0")
	if v.Major() != 0 || v.Minor() != 0 || v.Patch() != 0 {
		t.Error("0.0.0 should have all components zero")
	}
}
