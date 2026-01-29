# Security Best Practices

Skillsync includes built-in detection of sensitive data patterns to help prevent accidental exposure of credentials, API keys, and other secrets when syncing skills across platforms.

## Overview

When you run `skillsync sync`, the tool automatically scans skill content for common sensitive data patterns before syncing. This helps protect against:

- Accidentally syncing API keys or tokens
- Exposing passwords or credentials
- Sharing AWS/cloud provider access keys
- Leaking private keys or certificates
- Revealing database connection strings with credentials

## Detected Patterns

Skillsync detects the following patterns with varying severity levels:

### High Severity (Errors)

These patterns are flagged as errors and represent significant security risks:

- **AWS Access Keys**: `AKIA[A-Z0-9]{16}`
  ```yaml
  aws_access_key_id: AKIAIOSFODNN7EXAMPLE  # ❌ Detected
  ```

- **AWS Secret Keys**: 40-character base64 strings
  ```yaml
  aws_secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY  # ❌ Detected
  ```

- **GitHub Personal Access Tokens**: `ghp_[A-Za-z0-9]{36,}`
  ```yaml
  github_token: ghp_1234567890123456789012345678901234abcd  # ❌ Detected
  ```

- **Private Keys**: RSA/SSH private key headers
  ```
  -----BEGIN RSA PRIVATE KEY-----  # ❌ Detected
  ```

- **Database Connection Strings**: URLs with credentials
  ```
  postgres://user:password@host:5432/db  # ❌ Detected
  mysql://admin:secret@db.example.com/mydb  # ❌ Detected
  ```

### Medium Severity (Warnings)

These patterns are flagged as warnings and should be reviewed:

- **API Keys**: Various formats (api_key, API_KEY, apikey)
  ```yaml
  api_key: sk_test_1234567890123456  # ⚠️ Warning
  ```

- **Tokens**: Authentication tokens
  ```yaml
  token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...  # ⚠️ Warning
  auth_token: Bearer.eyJ...  # ⚠️ Warning
  ```

- **Passwords**: Password fields with values
  ```yaml
  password: MySecureP@ssw0rd  # ⚠️ Warning
  ```

- **Generic Secrets**: Secret keys and values
  ```yaml
  secret: my-secret-value-1234567890  # ⚠️ Warning
  secret_key: super-secret-key-123456  # ⚠️ Warning
  ```

- **Bearer Tokens**: Authorization headers
  ```
  Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...  # ⚠️ Warning
  ```

## False Positive Prevention

Skillsync is designed to minimize false positives by skipping:

### 1. Comments
```yaml
# api_key: your_api_key_here  # ✅ Ignored (comment)
// token: your_token_here      # ✅ Ignored (comment)
```

### 2. Documentation Examples
```yaml
api_key: <your_api_key_here>   # ✅ Ignored (placeholder)
token: example_token           # ✅ Ignored (example keyword)
password: your_password_here   # ✅ Ignored (your_ prefix)
secret: xxxxxxxxxxxxx          # ✅ Ignored (xxx pattern)
```

### 3. Placeholders
Common placeholder patterns are recognized and ignored:
- `<your_*>` or `<YOUR_*>`
- `example*` or `placeholder*`
- `your_*` prefixes
- Sequences of `x` characters

## Remediation Strategies

If sensitive data is detected in your skills, consider these approaches:

### 1. Use Environment Variables

Instead of hardcoding secrets:
```yaml
# ❌ Bad
api_key: sk_test_1234567890123456

# ✅ Good
api_key: ${API_KEY}
# or reference environment variable in instructions
# Set your API key via: export API_KEY=your_key_here
```

### 2. Use Configuration Files (Gitignored)

Store secrets in local config files that are excluded from version control:
```bash
# In your skill:
# Load API key from ~/.config/myapp/credentials.yaml
# (ensure this file is in .gitignore)
```

### 3. Reference External Secret Management

Document where to obtain secrets without including them:
```markdown
# Setup Instructions

1. Obtain your API key from https://example.com/api-keys
2. Set the environment variable: `export MY_API_KEY=<your_key>`
3. Run the skill
```

### 4. Use Templating

For skills that need to be customized, use template placeholders:
```yaml
# Template for user to fill in:
api_key: YOUR_API_KEY_HERE
password: YOUR_PASSWORD_HERE
```

## Security Validation Output

When running `skillsync sync`, you'll see security validation results:

```
Validating source skills...
  Found 5 valid skill(s)
Scanning for sensitive data...
  ⚠ Warning: API key pattern detected at line 12 (in deployment-script)
  ✗ Error: AWS access key detected at line 34 (in skill:aws-deploy:content)
  ⚠ Warning: Password pattern detected at line 8 (in database-setup)

Found 3 potential sensitive data issue(s).
Review the warnings above before syncing these skills.
Consider removing sensitive data or using environment variables instead.
```

### Understanding the Output

- **Line numbers**: Pinpoint exactly where the issue was detected
- **Skill name**: Identifies which skill contains the issue
- **Pattern type**: Describes what kind of sensitive data was found
- **Severity**:
  - `⚠ Warning`: Review recommended, may be a false positive
  - `✗ Error`: High-confidence detection, action strongly recommended

## Best Practices Summary

1. **Never commit secrets**: Use environment variables or secret management tools
2. **Review warnings**: Even if a warning seems like a false positive, consider if it could be improved
3. **Use placeholders in templates**: Make it obvious what needs to be replaced
4. **Document secret sources**: Tell users where to obtain credentials without including them
5. **Regular audits**: Periodically review your skills for sensitive data
6. **Use `.gitignore`**: Ensure credential files are excluded from version control
7. **Rotate exposed secrets**: If you accidentally commit secrets, rotate them immediately

## Configuration

Currently, sensitive data detection is always enabled during sync operations. Configuration options may be added in future releases to:

- Customize detection patterns
- Adjust severity levels
- Skip specific patterns
- Add custom patterns

## Reporting Issues

If you encounter:
- **False positives**: Patterns incorrectly flagged as sensitive
- **Missed patterns**: Sensitive data that should be detected
- **Improvements**: Better ways to handle specific cases

Please report them at: https://github.com/klauern/skillsync/issues

## Additional Resources

- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [GitHub: Removing sensitive data](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [AWS: Best practices for managing access keys](https://docs.aws.amazon.com/general/latest/gr/aws-access-keys-best-practices.html)
