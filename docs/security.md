# Security Best Practices

Skillsync includes comprehensive security features to protect your skills and credentials. This guide covers sensitive data detection, permission models, file security, multi-user considerations, and incident response.

## Table of Contents

- [Overview](#overview)
- [Sensitive Data Detection](#sensitive-data-detection)
- [Permission Model](#permission-model)
- [File Permissions & Multi-User Systems](#file-permissions--multi-user-systems)
- [Sync Security Implications](#sync-security-implications)
- [Incident Response](#incident-response)
- [Best Practices Summary](#best-practices-summary)
- [Additional Resources](#additional-resources)

## Overview

When you run `skillsync sync`, the tool automatically:

1. **Scans skill content** for sensitive data patterns (API keys, tokens, credentials)
2. **Validates permissions** for destructive operations (delete, overwrite)
3. **Checks file permissions** and scope access controls
4. **Creates backups** before destructive operations

This multi-layered approach helps protect against:

- Accidentally syncing API keys or tokens
- Exposing passwords or credentials
- Sharing AWS/cloud provider access keys
- Leaking private keys or certificates
- Revealing database connection strings with credentials
- Unauthorized destructive operations
- Data loss from overwrites or deletions

## Sensitive Data Detection

### Detected Patterns

When running `skillsync sync`, the tool automatically scans all skill content for common sensitive data patterns:

#### High Severity (Errors)

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

#### Medium Severity (Warnings)

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

### False Positive Prevention

Skillsync is designed to minimize false positives by skipping:

#### 1. Comments
```yaml
# api_key: your_api_key_here  # ✅ Ignored (comment)
// token: your_token_here      # ✅ Ignored (comment)
```

#### 2. Documentation Examples
```yaml
api_key: <your_api_key_here>   # ✅ Ignored (placeholder)
token: example_token           # ✅ Ignored (example keyword)
password: your_password_here   # ✅ Ignored (your_ prefix)
secret: xxxxxxxxxxxxx          # ✅ Ignored (xxx pattern)
```

#### 3. Placeholders
Common placeholder patterns are recognized and ignored:
- `<your_*>` or `<YOUR_*>`
- `example*` or `placeholder*`
- `your_*` prefixes
- Sequences of `x` characters

### Remediation Strategies

If sensitive data is detected in your skills, consider these approaches:

#### 1. Use Environment Variables

Instead of hardcoding secrets:
```yaml
# ❌ Bad
api_key: sk_test_1234567890123456

# ✅ Good
api_key: ${API_KEY}
# or reference environment variable in instructions
# Set your API key via: export API_KEY=your_key_here
```

#### 2. Use Configuration Files (Gitignored)

Store secrets in local config files that are excluded from version control:
```bash
# In your skill:
# Load API key from ~/.config/myapp/credentials.yaml
# (ensure this file is in .gitignore)
```

#### 3. Reference External Secret Management

Document where to obtain secrets without including them:
```markdown
# Setup Instructions

1. Obtain your API key from https://example.com/api-keys
2. Set the environment variable: `export MY_API_KEY=<your_key>`
3. Run the skill
```

#### 4. Use Templating

For skills that need to be customized, use template placeholders:
```yaml
# Template for user to fill in:
api_key: YOUR_API_KEY_HERE
password: YOUR_PASSWORD_HERE
```

### Security Validation Output

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

#### Understanding the Output

- **Line numbers**: Pinpoint exactly where the issue was detected
- **Skill name**: Identifies which skill contains the issue
- **Pattern type**: Describes what kind of sensitive data was found
- **Severity**:
  - `⚠ Warning`: Review recommended, may be a false positive
  - `✗ Error`: High-confidence detection, action strongly recommended

### Configuration

Sensitive data detection can be configured in `~/.skillsync/config.yaml` under the `security.detection` section.

#### Basic Configuration

```yaml
security:
  detection:
    enabled: true  # Enable/disable detection (default: true)
```

#### Disabling Specific Patterns

Disable built-in patterns by name:

```yaml
security:
  detection:
    enabled: true
    disabled_patterns:
      - "API Key"
      - "Generic Secret"
```

#### Custom Patterns

Add organization-specific patterns:

```yaml
security:
  detection:
    enabled: true
    custom_patterns:
      - name: "Internal API Token"
        regex: "int_[a-zA-Z0-9]{32}"
        description: "Internal API token detected"
        severity: "error"  # "error" or "warning"

      - name: "OAuth Client Secret"
        regex: "(?i)client[_-]?secret\\s*[:=]\\s*['\"]?[a-zA-Z0-9_-]{20,}['\"]?"
        description: "OAuth client secret detected"
        severity: "warning"
```

#### Pattern Overrides

Change severity levels for built-in patterns:

```yaml
security:
  detection:
    enabled: true
    pattern_overrides:
      "Password":
        severity: "error"  # Upgrade from warning to error
      "Bearer Token":
        severity: "disabled"  # Disable this pattern
```

#### Skill Exceptions

Skip detection for specific skills:

```yaml
security:
  detection:
    enabled: true
    skill_exceptions:
      - "test-credentials.md"
      - "auth-examples.md"
```

#### Complete Example

```yaml
security:
  detection:
    enabled: true

    # Disable patterns not relevant to your workflow
    disabled_patterns:
      - "Generic Secret"

    # Add custom patterns for your organization
    custom_patterns:
      - name: "Acme Corp API Key"
        regex: "acme_[a-zA-Z0-9]{24}"
        description: "Acme Corp API key detected"
        severity: "error"

    # Adjust severity levels
    pattern_overrides:
      "Password":
        severity: "error"  # More strict
      "API Key":
        severity: "warning"  # Less strict

    # Skip detection for test/documentation skills
    skill_exceptions:
      - "test-auth.md"
      - "api-examples.md"
```

#### Environment Variables

Security settings can be overridden with environment variables:

```bash
# Disable detection
export SKILLSYNC_SECURITY_DETECTION_ENABLED=false

# Disable specific patterns (comma-separated)
export SKILLSYNC_SECURITY_DETECTION_DISABLED_PATTERNS="API Key,Password"

# Skip detection for specific skills (comma-separated)
export SKILLSYNC_SECURITY_DETECTION_SKILL_EXCEPTIONS="test.md,examples.md"
```

#### Built-in Patterns

The following patterns are available for configuration:

**High Severity (Errors)**:
- `AWS Access Key`
- `AWS Secret Key`
- `GitHub Token`
- `Private Key`
- `Database Connection String`

**Medium Severity (Warnings)**:
- `API Key`
- `Token`
- `Password`
- `Generic Secret`
- `Bearer Token`

## Permission Model

Skillsync uses a hierarchical permission system to control destructive operations and scope-level access. This prevents accidental data loss and unauthorized modifications.

### Permission Levels

Three permission levels control what operations are allowed:

| Level | Operations Allowed | Use Case |
|-------|-------------------|----------|
| `read-only` | list, show, compare | Viewing skills only, no modifications |
| `write` | sync, add, promote/demote, backup | Standard operations, no destructive actions |
| `destructive` | delete, overwrite, backup deletion | Full control including data removal |

**Default**: `destructive` (backward compatible, all operations allowed)

### Operation Types

Each operation requires a minimum permission level:

| Operation | Required Level | Description |
|-----------|---------------|-------------|
| `read` | read-only | View skill content |
| `write` | write | Add or sync skills (creates backups) |
| `backup` | write | Create manual backups |
| `delete` | destructive | Remove skills permanently |
| `overwrite` | destructive | Replace existing skills without prompting |
| `backup_delete` | destructive | Delete backup files |

### Scope Permissions

Control which storage scopes can be modified:

```yaml
# ~/.skillsync/permissions.yaml
scope_permissions:
  allow_user_scope: true     # ~/.{platform}/skills/ (default: true)
  allow_repo_scope: true     # .{platform}/skills/ in repo (default: true)
  allow_system_scope: false  # /etc, /opt paths (default: false)
```

**Scope hierarchy:**
1. **Repo Scope** (highest priority): `.claude/skills/`, `.cursor/skills/`, `.codex/skills/` in project root
2. **User Scope**: `~/.claude/skills/`, `~/.cursor/skills/`, etc. in home directory
3. **Admin Scope**: `/opt/{platform}/skills/` (custom paths)
4. **System Scope**: `/etc/{platform}/skills/` (custom paths)

⚠️ **Warning**: System and admin scope writes are **disabled by default** to prevent accidental modifications to shared system skills.

### Confirmation Requirements

Configure which operations require user confirmation:

```yaml
# ~/.skillsync/permissions.yaml
require_confirmation:
  delete: true                    # Always prompt before deleting skills
  overwrite: false                # Auto-backup handles safety
  backup_delete: true             # Always confirm backup deletion
  promote_with_removal: true      # Confirm when promoting moves skills
```

When confirmation is required, you'll see:

```
⚠️  Destructive Operation: Delete
This will permanently remove 3 skill(s) from user scope.
This operation cannot be undone (no backup will be created).

Files to be deleted:
  - ~/.claude/skills/git-commit-helper.md
  - ~/.claude/skills/api-request-builder.md
  - ~/.claude/skills/test-generator.md

Continue? [y/N]:
```

### Permission Configuration

Create or edit `~/.skillsync/permissions.yaml`:

```yaml
# Permission level (read-only, write, destructive)
permission_level: write

# Operation-specific permissions
operation_permissions:
  read: true
  write: true
  backup: true
  delete: false      # Disable delete operations
  overwrite: false   # Disable overwrites
  backup_delete: false

# Scope access control
scope_permissions:
  allow_user_scope: true
  allow_repo_scope: true
  allow_system_scope: false

# Confirmation requirements
require_confirmation:
  delete: true
  overwrite: false
  backup_delete: true
  promote_with_removal: true
```

### Example Scenarios

#### Scenario 1: Read-Only Mode (CI/CD)

For automated environments where skills should only be read:

```yaml
permission_level: read-only
```

All write, backup, and delete operations will be blocked.

#### Scenario 2: Safe Mode (Prevent Accidental Deletions)

Allow sync but prevent destructive operations:

```yaml
permission_level: write
operation_permissions:
  delete: false
  overwrite: false
  backup_delete: false
```

#### Scenario 3: Repo-Only Mode

Only allow modifications to repository-level skills:

```yaml
permission_level: destructive
scope_permissions:
  allow_user_scope: false   # Block user scope writes
  allow_repo_scope: true
  allow_system_scope: false
```

## File Permissions & Multi-User Systems

Understanding file permissions is critical for security on shared systems.

### File Permission Overview

Skillsync creates files and directories with specific Unix permissions:

| Component | Location | Permission | Readable By |
|-----------|----------|-----------|-------------|
| **Skill Files** | `.{platform}/skills/` | `0o644` (rw-r--r--) | All users |
| **User Skills** | `~/.{platform}/skills/` | `0o644` (rw-r--r--) | All users (home dir provides isolation) |
| **Config Files** | `~/.skillsync/config.yaml` | `0o644` (rw-r--r--) | All users (home dir provides isolation) |
| **Backup Files** | `~/.skillsync/backups/` | `0o640` (rw-r-----) | Owner + group |
| **Directories** | All | `0o750` (rwxr-x---) | Owner + group (execute) |

### Security Implications

#### On Single-User Systems
- File permissions are sufficient (home directory isolation)
- Skills in `~/.{platform}/skills/` are only accessible by the user
- Repo skills in `.{platform}/skills/` are project-specific

#### On Multi-User Systems
⚠️ **Important**: Skills are **world-readable by default** (0o644).

**Risks:**
1. **Repository Skills**: Any user with access to a shared repository can read all skills in `.{platform}/skills/`
2. **System Skills**: If using system or admin scopes, skills are readable by all users on the system
3. **Config Exposure**: While stored in user home directories, config files could expose skill locations

**Mitigation Strategies:**

1. **Restrict Directory Permissions** (Recommended for shared systems):
   ```bash
   # Make ~/.skillsync and skill directories private
   chmod 0700 ~/.skillsync
   chmod 0700 ~/.claude
   chmod 0700 ~/.cursor
   chmod 0700 ~/.codex

   # Make skill files private
   find ~/.claude/skills -type f -exec chmod 0600 {} \;
   find ~/.cursor/skills -type f -exec chmod 0600 {} \;
   ```

2. **Use Environment Variables** for secrets instead of embedding them in skills

3. **Avoid System/Admin Scopes** on multi-user systems unless absolutely necessary

4. **Repository Skills**: Be aware that repo skills (`.claude/skills/`) are shared with all repo collaborators

### Multi-User Considerations

#### User Isolation
- **User Scope**: Each user has isolated skills in their home directory (`~/.{platform}/skills/`)
- **No Per-User Filtering**: Skills in repo scope (`.{platform}/skills/`) are accessible to all users with repo access

#### Repository Collaboration
When multiple users work in the same repository:
- All users can read/modify repo-scope skills (`.claude/skills/`, etc.)
- Changes to repo skills affect all team members
- Consider this when deciding whether to use repo scope vs user scope

#### Backup Considerations
- Backups are stored in `~/.skillsync/backups/` (user-private location)
- Backup files use more restrictive permissions (0o640)
- Backups may contain sensitive data from skills

### Recommended Setup for Shared Systems

```bash
# 1. Create skill directories with restrictive permissions
mkdir -p ~/.claude/skills
mkdir -p ~/.cursor/skills
mkdir -p ~/.codex/skills
chmod 0700 ~/.claude ~/.cursor ~/.codex

# 2. Set up skillsync config directory
mkdir -p ~/.skillsync/backups
chmod 0700 ~/.skillsync

# 3. Configure permissions in skillsync
cat > ~/.skillsync/permissions.yaml <<EOF
permission_level: write
scope_permissions:
  allow_user_scope: true
  allow_repo_scope: true
  allow_system_scope: false  # Never allow on shared systems
require_confirmation:
  delete: true
  overwrite: false
  backup_delete: true
EOF

# 4. Verify permissions
ls -la ~/.claude
ls -la ~/.skillsync
```

## Sync Security Implications

Understanding the security implications of syncing skills across platforms.

### Data at Rest

**Storage Format:**
- Skills are stored as **plaintext files** (Markdown, JSON)
- **No encryption at rest** by default
- Config files stored as **plaintext YAML**
- Backups stored **unencrypted**

**Security Considerations:**
- Any secrets in skills are stored in plaintext
- Skills in repo scope (`.{platform}/skills/`) are committed to version control
- Backups in `~/.skillsync/backups/` contain full skill content

**Recommendations:**
- Use sensitive data detection to catch secrets before syncing
- Store secrets externally (environment variables, secret managers)
- Use `.gitignore` to exclude skills with sensitive data from version control

### Data in Transit

**Syncing Between Platforms:**
- Skillsync reads from source and writes to target
- No network transfer by default (local filesystem operations)
- If syncing to/from network filesystems (NFS, SMB), data is subject to network security

**Version Control:**
- If using repo scope, skills are committed to git
- Skills are pushed/pulled with repository
- Consider using git-crypt or similar tools for encrypted repo storage

### Backup Security

**Automatic Backups:**
- Created before overwrite or delete operations
- Stored in `~/.skillsync/backups/` with permissions 0o640
- Named using SHA256 hash of original content
- Metadata stored in `~/.skillsync/metadata/`

**Backup Retention:**
```yaml
# ~/.skillsync/config.yaml
backup:
  enabled: true
  retention_days: 30    # Backups older than 30 days are eligible for cleanup
  max_backups: 10       # Keep at most 10 backups per skill
```

**Security Implications:**
- Backups may contain secrets even after they're removed from active skills
- Old backups accumulate sensitive data
- Backup deletion requires destructive permission level

**Best Practices:**
1. Regularly clean up old backups: `skillsync backup clean --dry-run` (check first)
2. Set reasonable retention limits (default: 30 days, 10 backups)
3. If a secret is exposed, delete backups containing it:
   ```bash
   # Find backups containing exposed secret
   grep -r "exposed_secret_pattern" ~/.skillsync/backups/

   # Delete specific backup (requires destructive permission)
   skillsync backup delete <backup-id>
   ```

### Cross-Platform Security

**Platform-Specific Considerations:**

| Platform | Skill Format | Location | Security Notes |
|----------|-------------|----------|----------------|
| Claude Code | `.md`, `.txt`, or no extension | `.claude/skills/` | Markdown can embed scripts |
| Cursor | `.md`, `.mdc` | `.cursor/skills/` | `.mdc` format may have platform-specific features |
| Codex | `.json` | `.codex/skills/` | JSON may contain code snippets |

**Validation:**
- Skillsync validates file extensions per platform
- Format validation prevents incompatible files from syncing
- Content validation detects sensitive data patterns

**Risk:** Skills containing scripts or code could execute in AI assistant context if not carefully reviewed.

## Incident Response

What to do if sensitive data is exposed in skills.

### If You Detect Secrets in Skills

#### Immediate Actions (Within Minutes)

1. **Stop Syncing:**
   ```bash
   # Set permissions to read-only to prevent further propagation
   skillsync config set permission_level read-only
   ```

2. **Identify Exposure Scope:**
   ```bash
   # Check which skills contain the secret
   grep -r "exposed_secret_pattern" ~/.claude/skills/
   grep -r "exposed_secret_pattern" ~/.cursor/skills/
   grep -r "exposed_secret_pattern" ~/.codex/skills/

   # Check repository skills
   grep -r "exposed_secret_pattern" .claude/skills/
   grep -r "exposed_secret_pattern" .cursor/skills/

   # Check backups
   grep -r "exposed_secret_pattern" ~/.skillsync/backups/
   ```

3. **Rotate Credentials Immediately:**
   - AWS keys: [Rotate access keys](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_RotateAccessKey)
   - GitHub tokens: Regenerate at https://github.com/settings/tokens
   - API keys: Contact service provider to revoke/regenerate
   - Database credentials: Update passwords immediately

#### Short-Term Actions (Within Hours)

4. **Remove Secrets from Skills:**
   ```bash
   # Edit skills to remove secrets
   $EDITOR ~/.claude/skills/affected-skill.md

   # Replace with environment variable references
   # Before: api_key: sk_live_abc123
   # After:  api_key: ${API_KEY}
   ```

5. **Delete Compromised Backups:**
   ```bash
   # Find backups created during exposure window
   skillsync backup list

   # Delete specific backups (requires destructive permission)
   skillsync backup delete <backup-id>
   ```

6. **Check Version Control History:**
   ```bash
   # If using repo scope, check git history
   git log -p -- .claude/skills/affected-skill.md

   # If secret was committed, see GitHub's guide:
   # https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository
   ```

#### Medium-Term Actions (Within Days)

7. **Audit Related Credentials:**
   - Check for other credentials in same service
   - Review access logs for unauthorized access
   - Update any secrets shared with compromised credentials

8. **Review Security Practices:**
   - Run sensitive data detection on all skills:
     ```bash
     skillsync sync --dry-run
     # Review any warnings or errors
     ```
   - Update skills to use environment variables
   - Enable stricter permission settings

9. **Update Team (If Applicable):**
   - Notify team members if using repo scope
   - Document incident and remediation steps
   - Update team security guidelines

#### Long-Term Actions (Ongoing)

10. **Implement Preventive Measures:**
    - Enable pre-commit hooks for sensitive data detection
    - Set up regular security audits:
      ```bash
      # Weekly audit script
      #!/bin/bash
      skillsync sync --dry-run | grep -E "⚠|✗"
      ```
    - Use external secret management (AWS Secrets Manager, HashiCorp Vault)
    - Document secret handling procedures for the team

11. **Monitor for Abuse:**
    - Check service logs for unusual activity
    - Set up alerts for API usage anomalies
    - Review access patterns for compromised credentials

### If Secrets Are in Version Control

If secrets were committed to git repository (repo scope skills):

1. **Use git-filter-repo or BFG Repo-Cleaner:**
   ```bash
   # Install git-filter-repo
   pip install git-filter-repo

   # Remove sensitive file from history
   git filter-repo --path .claude/skills/affected-skill.md --invert-paths

   # Or use BFG
   bfg --delete-files affected-skill.md
   ```

2. **Force Push (Coordinate with Team):**
   ```bash
   git push --force-with-lease origin main
   ```

3. **Notify Team to Rebase:**
   ```bash
   # Team members should fetch and reset
   git fetch origin
   git reset --hard origin/main
   ```

4. **Rotate Credentials** (even if removed from history, consider them compromised)

### Reporting Security Issues

If you discover a security vulnerability in skillsync itself:

1. **Do NOT open a public GitHub issue**
2. **Contact maintainers privately**: security@skillsync.dev (or see SECURITY.md in repo)
3. **Provide details**: Steps to reproduce, impact assessment, suggested fixes

## Best Practices Summary

### Sensitive Data Protection
1. **Never commit secrets**: Use environment variables or secret management tools instead
2. **Review warnings**: Even warnings that seem like false positives should be investigated
3. **Use placeholders in templates**: Make it obvious what needs to be replaced (e.g., `YOUR_API_KEY_HERE`)
4. **Document secret sources**: Tell users where to obtain credentials without including them
5. **Regular audits**: Periodically run `skillsync sync --dry-run` to scan for sensitive data
6. **Use `.gitignore`**: Ensure credential files are excluded from version control
7. **Rotate exposed secrets immediately**: If you accidentally expose secrets, rotate them within minutes

### Permission Management
8. **Use appropriate permission levels**: Start with `write` level, only enable `destructive` when needed
9. **Enable confirmations**: Require confirmation for `delete` and `backup_delete` operations
10. **Disable system scope**: Never enable `allow_system_scope` on shared systems
11. **Review scope permissions**: Understand which scopes (user, repo, system) are writable

### File Security (Multi-User Systems)
12. **Restrict directory permissions**: Use `chmod 0700` on skill directories for shared systems
13. **Use user scope for personal skills**: Reserve repo scope for team-shared skills only
14. **Be aware of world-readable files**: Skills are 0o644 by default (readable by all users)
15. **Secure backup directory**: Ensure `~/.skillsync/` has restricted permissions

### Sync Security
16. **Understand data at rest**: Skills and backups are stored unencrypted
17. **Clean old backups**: Regularly remove backups with `skillsync backup clean`
18. **Review repo scope carefully**: Skills in `.{platform}/skills/` are committed to version control
19. **Use git-crypt for sensitive repos**: Consider encrypted git storage if repo skills contain configuration

### Incident Response Preparedness
20. **Know your exposure scope**: Understand which scopes contain skills (user, repo, system)
21. **Have rotation procedures ready**: Document how to rotate each type of credential you use
22. **Test backup restoration**: Ensure you can restore skills from backups if needed
23. **Monitor service logs**: Set up alerts for unusual API usage after credential exposure

## Additional Resources

### Security Guides
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)

### Credential Rotation
- [GitHub: Removing sensitive data from repositories](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [AWS: Best practices for managing access keys](https://docs.aws.amazon.com/general/latest/gr/aws-access-keys-best-practices.html)
- [GitHub: Token security best practices](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/token-security-best-practices)

### Secret Management Tools
- [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/)
- [HashiCorp Vault](https://www.vaultproject.io/)
- [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/)
- [Google Cloud Secret Manager](https://cloud.google.com/secret-manager)

### Git Security
- [git-filter-repo](https://github.com/newren/git-filter-repo) - Remove secrets from git history
- [BFG Repo-Cleaner](https://rtyley.github.io/bfg-repo-cleaner/) - Alternative tool for cleaning repos
- [git-crypt](https://github.com/AGWA/git-crypt) - Transparent file encryption in git

### File Permissions
- [Linux File Permissions Guide](https://wiki.archlinux.org/title/File_permissions_and_attributes)
- [Understanding Unix Permissions](https://www.redhat.com/sysadmin/linux-file-permissions-explained)

## Reporting Issues

### Security Vulnerabilities
If you discover a security vulnerability in skillsync:
- **Do NOT open a public issue**
- Contact: security@skillsync.dev (or see SECURITY.md)
- Provide: Steps to reproduce, impact assessment, suggested fixes

### False Positives or Missed Patterns
If sensitive data detection has issues:
- **False positives**: Patterns incorrectly flagged as sensitive
- **Missed patterns**: Sensitive data that should be detected
- **Improvements**: Better ways to handle specific cases

Please report at: https://github.com/klauern/skillsync/issues

---

**Related Documentation:**
- [Quick Start Guide](quick-start.md) - Initial setup and configuration
- [Commands Reference](commands.md) - Detailed command documentation
- [Sync Strategies](sync-strategies.md) - Understanding sync behavior
