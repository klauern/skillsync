---
name: cursor-complex-globs
description: Cursor skill with complex glob patterns
globs:
  - "src/**/*.{ts,tsx,js,jsx}"
  - "tests/**/*.test.ts"
  - "!**/*.spec.ts"
  - "**/components/**"
alwaysApply: false
---
# Cursor Complex Globs

This Cursor skill demonstrates complex glob patterns including:
- Multiple file extensions
- Subdirectory wildcards
- Exclusion patterns
- Path-specific matching
