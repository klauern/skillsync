package security

import (
	"strings"
	"testing"
)

func TestScanContent_Clean(t *testing.T) {
	content := `# Clean Skill
This is a clean skill with no sensitive data.
Just normal content here.`

	result := ScanContent(content)

	if !result.Valid {
		t.Errorf("Expected valid result for clean content, got: %v", result.Errors)
	}
	if len(result.Warnings) > 0 {
		t.Errorf("Expected no warnings for clean content, got: %v", result.Warnings)
	}
}

func TestScanContent_APIKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "api_key lowercase",
			content: "api_key: sk_test_1234567890123456",
		},
		{
			name:    "API_KEY uppercase",
			content: "API_KEY=sk_test_1234567890123456",
		},
		{
			name:    "apiKey camelCase",
			content: `apiKey: "sk_test_1234567890123456"`,
		},
		{
			name:    "api-key with dash",
			content: "api-key = sk_test_1234567890123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			// Warnings don't invalidate the result, they just warn
			if !result.Valid {
				t.Error("Expected valid result (warnings don't invalidate)")
			}
			if len(result.Warnings) == 0 {
				t.Error("Expected warnings for API key detection")
			}
			if !strings.Contains(result.Warnings[0], "API key") {
				t.Errorf("Expected warning about API key, got: %s", result.Warnings[0])
			}
		})
	}
}

func TestScanContent_Token(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "token",
			content: "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:    "access_token",
			content: "access_token=ya29.a0AfH6SMBx...",
		},
		{
			name:    "auth_token",
			content: `auth_token: "Bearer.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			// Warnings don't invalidate the result
			if !result.Valid {
				t.Error("Expected valid result (warnings don't invalidate)")
			}
			if len(result.Warnings) == 0 {
				t.Error("Expected warnings for token detection")
			}
			if !strings.Contains(result.Warnings[0], "token") {
				t.Errorf("Expected warning about token, got: %s", result.Warnings[0])
			}
		})
	}
}

func TestScanContent_Password(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "password",
			content: "password: MySecureP@ssw0rd",
		},
		{
			name:    "passwd",
			content: "passwd=SecretPassword123",
		},
		{
			name:    "pwd",
			content: `pwd: "P@ssw0rd!"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			// Warnings don't invalidate the result
			if !result.Valid {
				t.Error("Expected valid result (warnings don't invalidate)")
			}
			if len(result.Warnings) == 0 {
				t.Error("Expected warnings for password detection")
			}
			if !strings.Contains(result.Warnings[0], "Password") {
				t.Errorf("Expected warning about password, got: %s", result.Warnings[0])
			}
		})
	}
}

func TestScanContent_AWSAccessKey(t *testing.T) {
	content := `AWS configuration:
aws_access_key_id: AKIAIOSFODNN7EXAMPLE`

	result := ScanContent(content)

	if result.Valid {
		t.Error("Expected invalid result for AWS key content")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for AWS key detection (severity: error)")
	}
	// AWS keys should be errors, not warnings
	if len(result.Warnings) > 0 {
		t.Errorf("Expected errors not warnings for AWS keys")
	}
}

func TestScanContent_AWSSecretKey(t *testing.T) {
	content := `AWS configuration:
aws_secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

	result := ScanContent(content)

	if result.Valid {
		t.Error("Expected invalid result for AWS secret content")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for AWS secret detection (severity: error)")
	}
}

func TestScanContent_GitHubToken(t *testing.T) {
	content := `GitHub Actions:
github_token: ghp_1234567890123456789012345678901234abcd`

	result := ScanContent(content)

	if result.Valid {
		t.Error("Expected invalid result for GitHub token content")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for GitHub token detection (severity: error)")
	}
}

func TestScanContent_PrivateKey(t *testing.T) {
	content := `SSH Key:
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`

	result := ScanContent(content)

	if result.Valid {
		t.Error("Expected invalid result for private key content")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for private key detection (severity: error)")
	}
}

func TestScanContent_GenericSecret(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "secret",
			content: "secret: my-secret-value-1234567890",
		},
		{
			name:    "secret_key",
			content: "secret_key=super-secret-key-123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			// Warnings don't invalidate the result
			if !result.Valid {
				t.Error("Expected valid result (warnings don't invalidate)")
			}
			if len(result.Warnings) == 0 {
				t.Error("Expected warnings for secret detection")
			}
		})
	}
}

func TestScanContent_BearerToken(t *testing.T) {
	content := `Authorization header:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0`

	result := ScanContent(content)

	// Warnings don't invalidate the result
	if !result.Valid {
		t.Error("Expected valid result (warnings don't invalidate)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for bearer token detection")
	}
}

func TestScanContent_DatabaseConnectionString(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "postgres",
			content: "DB_URL: postgres://user:password@localhost:5432/db",
		},
		{
			name:    "mysql",
			content: "DB_URL: mysql://admin:secret@db.example.com:3306/mydb",
		},
		{
			name:    "mongodb",
			content: "DB_URL: mongodb://dbuser:dbpass@mongo.example.com:27017/database",
		},
		{
			name:    "redis",
			content: "DB_URL: redis://user:pass@redis.example.com:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			if result.Valid {
				t.Error("Expected invalid result for database connection string")
			}
			if len(result.Errors) == 0 {
				t.Error("Expected errors for database connection string (severity: error)")
			}
		})
	}
}

func TestScanContent_FalsePositives(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "comment",
			content: `# Example configuration
# api_key: your_api_key_here
# token: your_token_here`,
		},
		{
			name: "documentation",
			content: `Set your API key:
api_key: <your_api_key_here>
token: example_token_placeholder`,
		},
		{
			name: "placeholder",
			content: `Configuration:
api_key: YOUR_API_KEY
password: xxxxxxxxxxxxx`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanContent(tt.content)

			if !result.Valid {
				t.Errorf("Expected valid result for false positive: %s, got errors: %v", tt.name, result.Errors)
			}
			if len(result.Warnings) > 0 {
				t.Errorf("Expected no warnings for false positive: %s, got: %v", tt.name, result.Warnings)
			}
		})
	}
}

func TestScanContent_MultipleDetections(t *testing.T) {
	content := `Configuration file with multiple issues:
api_key: sk_test_1234567890123456
password: MySecurePassword123
aws_access_key_id: AKIAIOSFODNN7EXAMPLE
github_token: ghp_1234567890123456789012345678901234abcd`

	result := ScanContent(content)

	if result.Valid {
		t.Error("Expected invalid result for multiple sensitive patterns")
	}

	// Should have both errors and warnings
	if len(result.Errors) == 0 {
		t.Error("Expected errors for high-severity patterns (AWS, GitHub)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for medium-severity patterns (API key, password)")
	}

	// Should detect at least 4 patterns
	totalDetections := len(result.Errors) + len(result.Warnings)
	if totalDetections < 4 {
		t.Errorf("Expected at least 4 detections, got %d", totalDetections)
	}
}

func TestScanContent_LineNumbers(t *testing.T) {
	content := `Line 1: clean
Line 2: clean
Line 3: api_key: sk_test_1234567890123456
Line 4: clean
Line 5: password: SecurePass123`

	result := ScanContent(content)

	// Warnings don't invalidate the result
	if !result.Valid {
		t.Error("Expected valid result (warnings don't invalidate)")
	}

	// Check that line numbers are reported
	if len(result.Warnings) < 2 {
		t.Errorf("Expected 2 warnings, got %d", len(result.Warnings))
	}

	// Check that warnings contain line numbers
	foundLine3 := false
	foundLine5 := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "line 3") {
			foundLine3 = true
		}
		if strings.Contains(warning, "line 5") {
			foundLine5 = true
		}
	}

	if !foundLine3 {
		t.Error("Expected warning to mention line 3")
	}
	if !foundLine5 {
		t.Error("Expected warning to mention line 5")
	}
}

func TestScanContent_Empty(t *testing.T) {
	result := ScanContent("")

	if !result.Valid {
		t.Error("Expected valid result for empty content")
	}
	if len(result.Warnings) > 0 {
		t.Error("Expected no warnings for empty content")
	}
	if len(result.Errors) > 0 {
		t.Error("Expected no errors for empty content")
	}
}

func TestValidateSkillContent(t *testing.T) {
	content := `Skill with sensitive data:
api_key: sk_test_1234567890123456`

	result := ValidateSkillContent(content, "test-skill")

	// Warnings don't invalidate the result
	if !result.Valid {
		t.Error("Expected valid result (warnings don't invalidate)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings")
	}

	// Check that skill name is included in error field
	for _, err := range result.Errors {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "test-skill") {
			t.Errorf("Expected skill name in error, got: %s", errMsg)
		}
	}
}

func TestIsFalsePositive(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{"hash comment", "# api_key: test", true},
		{"double slash comment", "// token: test", true},
		{"multi-line comment start", "/* password: test", true},
		{"multi-line comment body", "* secret: test", true},
		{"example_ prefix in value", "api_key: example_key", true},
		{"placeholder in value", "token: placeholder", true},
		{"your_ prefix in value", "api_key: your_api_key", true},
		{"angle bracket placeholder", "password: <your_password>", true},
		{"xxx pattern in value", "secret: xxxxxxxxxxxxx", true},
		{"real value", "api_key: sk_test_1234567890", false},
		{"real AWS key with EXAMPLE suffix", "aws_key: AKIAIOSFODNN7EXAMPLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFalsePositive(tt.line)
			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.line, result)
			}
		})
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxLen   int
		expected string
	}{
		{
			name:     "short line",
			line:     "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "exact length",
			line:     "exactly 10",
			maxLen:   10,
			expected: "exactly 10",
		},
		{
			name:     "long line",
			line:     "this is a very long line that should be truncated",
			maxLen:   20,
			expected: "this is a very lo...",
		},
		{
			name:     "with whitespace",
			line:     "  spaces before and after  ",
			maxLen:   15,
			expected: "spaces befor...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLine(tt.line, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
