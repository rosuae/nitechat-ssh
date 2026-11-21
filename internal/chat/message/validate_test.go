package message

import (
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	valid := map[string]string{
		"alice":                                "alice",
		"Alice":                                "alice",
		"john.doe":                             "john.doe",
		"a_b-c":                                "a_b-c",
		"7":                                    "7",
		strings.Repeat("a", maxUsernameLength): strings.Repeat("a", maxUsernameLength),
	}
	for in, want := range valid {
		got, err := ValidateUsername(in)
		if err != nil || got != want {
			t.Errorf("ValidateUsername(%q) = %q, %v; want %q, nil", in, got, err, want)
		}
	}

	invalid := []string{
		"",
		"Andrei Eduard",     // contains a space
		"\x1b[2Jevil",       // ANSI escape
		"alice@AbCd",        // would look like an existing user's display name
		"\xd0\xb0lice",      // cyrillic "а" (U+0430) instead of latin "a"
		"ali\xe2\x80\x8bce", // zero-width space (U+200B)
		"-alice",            // must start with letter/digit
		".alice",
		"System", // reserved
		"root",
		strings.Repeat("a", maxUsernameLength+1),
	}
	for _, in := range invalid {
		if got, err := ValidateUsername(in); err == nil {
			t.Errorf("ValidateUsername(%q) = %q, nil; want error", in, got)
		}
	}
}

func TestValidateRoomName(t *testing.T) {
	if got, err := ValidateRoomName("General"); err != nil || got != "general" {
		t.Errorf("ValidateRoomName(General) = %q, %v", got, err)
	}
	for _, in := range []string{"", "my room", "\x1b[31mred", "room\xe2\x80\xae", strings.Repeat("r", maxRoomNameLength+1)} {
		if _, err := ValidateRoomName(in); err == nil {
			t.Errorf("ValidateRoomName(%q) succeeded; want error", in)
		}
	}
}

func TestSanitizeBody(t *testing.T) {
	cases := map[string]string{
		"hello world":                       "hello world",
		"héllo 世界 👋":                        "héllo 世界 👋",
		"\x1b[2Jclear":                      "[2Jclear",
		"csi\u009b31mred":                   "csi31mred",
		"del\x7f":                           "del",
		"bidi \xe2\x80\xaetxt.exe":          "bidi txt.exe", // U+202E right-to-left override
		"zero\xe2\x80\x8bwidth\xe2\x81\xa0": "zerowidth",    // U+200B, U+2060
		"tab\tbell\a":                       "tabbell",
	}
	for in, want := range cases {
		if got := SanitizeBody(in); got != want {
			t.Errorf("SanitizeBody(%q) = %q; want %q", in, got, want)
		}
	}
}
