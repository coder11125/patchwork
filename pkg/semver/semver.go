package semver

import (
	"fmt"

	masterminds "github.com/Masterminds/semver/v3"
)

type Version struct {
	v *masterminds.Version
}

func Parse(version string) (*Version, error) {
	v, err := masterminds.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("parse version %q: %w", version, err)
	}
	return &Version{v: v}, nil
}

func MustParse(version string) *Version {
	v, err := Parse(version)
	if err != nil {
		panic(err)
	}
	return v
}

func (v *Version) String() string {
	return v.v.Original()
}

func (v *Version) Major() uint64 {
	return v.v.Major()
}

func (v *Version) Minor() uint64 {
	return v.v.Minor()
}

func (v *Version) Patch() uint64 {
	return v.v.Patch()
}

func (v *Version) Prerelease() string {
	return v.v.Prerelease()
}

func (v *Version) Metadata() string {
	return v.v.Metadata()
}

func Compare(v1, v2 string) (int, error) {
	a, err := masterminds.NewVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("compare version v1 %q: %w", v1, err)
	}
	b, err := masterminds.NewVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("compare version v2 %q: %w", v2, err)
	}
	return a.Compare(b), nil
}

func IsNewer(current, latest string) (bool, error) {
	result, err := Compare(current, latest)
	if err != nil {
		return false, err
	}
	return result < 0, nil
}

func Satisfies(version, constraint string) (bool, error) {
	v, err := masterminds.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("satisfies version %q: %w", version, err)
	}
	c, err := masterminds.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Errorf("satisfies constraint %q: %w", constraint, err)
	}
	return c.Check(v), nil
}
