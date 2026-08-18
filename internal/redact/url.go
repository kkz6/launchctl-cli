package redact

import (
	"net/url"
	"strings"
)

// URL removes embedded credentials while retaining enough of the value to
// identify a repository or registry in terminal output.
func URL(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return redactUnparseableURL(value)
	}

	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if sensitiveQueryKey(key) {
			query.Set(key, "REDACTED")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func sensitiveQueryKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	switch key {
	case "auth", "authorization", "password", "passwd", "secret", "token", "access_token", "api_key", "credential", "credentials", "signature", "sig":
		return true
	}
	for _, suffix := range []string{"_password", "_secret", "_token", "_api_key", "_credential", "_signature"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func redactUnparseableURL(value string) string {
	if scheme := strings.Index(value, "://"); scheme >= 0 {
		rest := value[scheme+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			value = value[:scheme+3] + rest[at+1:]
		}
	}
	if question := strings.Index(value, "?"); question >= 0 {
		query := strings.ToLower(value[question+1:])
		for _, marker := range []string{"token=", "password=", "passwd=", "secret=", "auth=", "api_key=", "credential=", "signature="} {
			if strings.Contains(query, marker) {
				return value[:question] + "?REDACTED"
			}
		}
	}
	return value
}
