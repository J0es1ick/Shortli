package validator

import "testing"

func TestValidateShortCode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{name: "empty is optional", input: "", expected: ""},
		{name: "normalizes case and spaces", input: "  My-Link_7  ", expected: "my-link_7"},
		{name: "too short", input: "ab", shouldErr: true},
		{name: "too long", input: "abcdefghijklmnopqrstuvwxyz1234567", shouldErr: true},
		{name: "rejects unsupported characters", input: "my.link", shouldErr: true},
		{name: "rejects reserved alias", input: "API", shouldErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ValidateShortCode(test.input)
			if test.shouldErr {
				if err == nil {
					t.Fatalf("ValidateShortCode(%q) expected an error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateShortCode(%q) returned error: %v", test.input, err)
			}
			if actual != test.expected {
				t.Fatalf("ValidateShortCode(%q) = %q, want %q", test.input, actual, test.expected)
			}
		})
	}
}

func TestValidateURLRejectsUnsafeDestinations(t *testing.T) {
	tests := []string{
		"http://localhost/admin",
		"http://127.0.0.1/private",
		"http://192.168.1.12/device",
		"https://user:password@example.com",
		"file:///etc/passwd",
		"http://2130706433/private",
		"http://0x7f000001/private",
		"http://0177.0.0.1/private",
		"http://127.1/private",
	}
	for _, value := range tests {
		if _, err := ValidateURL(value); err == nil {
			t.Errorf("ValidateURL(%q) expected an error", value)
		}
	}
}

func TestValidateAccountCredentials(t *testing.T) {
	if email, err := ValidateEmail("  User@Example.com "); err != nil || email != "user@example.com" {
		t.Fatalf("ValidateEmail returned %q, %v", email, err)
	}
	for _, password := range []string{"short1", "onlyletters", "1234567890"} {
		if err := ValidatePassword(password); err == nil {
			t.Errorf("ValidatePassword(%q) expected an error", password)
		}
	}
	if err := ValidatePassword("strong-pass-42"); err != nil {
		t.Fatalf("ValidatePassword rejected a valid password: %v", err)
	}
}
