package redact

import "testing"

func TestURLRedactsCredentials(t *testing.T) {
	tests := map[string]string{
		"https://user:token@github.com/acme/api.git":                          "https://github.com/acme/api.git",
		"https://github.com/acme/api.git?access_token=secret&ref=main":        "https://github.com/acme/api.git?access_token=REDACTED&ref=main",
		"ssh://git:password@github.com/acme/api.git?signature=signed&depth=1": "ssh://github.com/acme/api.git?depth=1&signature=REDACTED",
		"git@github.com:acme/api.git":                                         "git@github.com:acme/api.git",
		"https://github.com/acme/api.git":                                     "https://github.com/acme/api.git",
		"https://user:token@github.com/%zz?access_token=secret":               "https://github.com/%zz?REDACTED",
	}

	for input, want := range tests {
		if got := URL(input); got != want {
			t.Errorf("URL(%q) = %q, want %q", input, got, want)
		}
	}
}
