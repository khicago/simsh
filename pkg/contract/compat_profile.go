package contract

import "fmt"

type CompatibilityProfile string

const (
	ProfileCoreStrict CompatibilityProfile = "core-strict"
	ProfileBashPlus   CompatibilityProfile = "bash-plus"
	ProfileZshLite    CompatibilityProfile = "zsh-lite"
)

type ProfileSpec struct {
	Name         CompatibilityProfile
	Capabilities map[string]bool
}

func DefaultProfile() CompatibilityProfile {
	return ProfileCoreStrict
}

func ParseProfile(raw string) (CompatibilityProfile, error) {
	profile := CompatibilityProfile(raw)
	switch profile {
	case "", ProfileCoreStrict:
		return ProfileCoreStrict, nil
	case ProfileBashPlus:
		return ProfileBashPlus, nil
	case ProfileZshLite:
		return ProfileZshLite, nil
	default:
		return "", fmt.Errorf("unsupported profile %q", raw)
	}
}

func ProfileByName(profile CompatibilityProfile) ProfileSpec {
	switch profile {
	case ProfileBashPlus:
		return ProfileSpec{Name: ProfileBashPlus, Capabilities: map[string]bool{"compat:date": true}}
	case ProfileZshLite:
		return ProfileSpec{Name: ProfileZshLite, Capabilities: map[string]bool{"compat:date": true, "glob:extended": true}}
	case ProfileCoreStrict:
		fallthrough
	default:
		return ProfileSpec{Name: ProfileCoreStrict, Capabilities: map[string]bool{}}
	}
}
