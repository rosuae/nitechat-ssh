package message

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const (
	maxUsernameLength = 24
	maxRoomNameLength = 32
)

// names are ASCII-only on purpose
var (
	usernamePattern = regexp.MustCompile(fmt.Sprintf(`^[a-z0-9][a-z0-9_.-]{0,%d}$`, maxUsernameLength-1))
	roomNamePattern = regexp.MustCompile(fmt.Sprintf(`^[a-z0-9][a-z0-9_.-]{0,%d}$`, maxRoomNameLength-1))
)

var reservedUsernames = map[string]struct{}{
	"system": {},
	"server": {},
	"admin":  {},
	"root":   {},
}

var UsernameRules = fmt.Sprintf(
	"Usernames must be 1-%d characters: lowercase letters, digits, '_', '-' or '.', starting with a letter or digit.",
	maxUsernameLength,
)

func ValidateUsername(raw string) (string, error) {
	name := strings.ToLower(raw)
	if !usernamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid username %q", raw)
	}
	if _, reserved := reservedUsernames[name]; reserved {
		return "", fmt.Errorf("username %q is reserved", raw)
	}
	return name, nil
}

func ValidateRoomName(raw string) (string, error) {
	name := strings.ToLower(raw)
	if !roomNamePattern.MatchString(name) {
		return "", fmt.Errorf(
			"invalid room name %q: use 1-%d characters: lowercase letters, digits, '_', '-' or '.'",
			raw, maxRoomNameLength,
		)
	}
	return name, nil
}

func SanitizeBody(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
}
