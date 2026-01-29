package template

// commandWrapperTemplate is the built-in template for command wrapper skills
const commandWrapperTemplate = `---
name: {{.Name}}
description: {{.Description}}
scope: {{.Scope}}
license: MIT
tools:{{range .Tools}}
  - {{.}}{{end}}
---

# {{.Name}}

A skill that wraps an external command or tool.

## Overview

This skill provides an interface to a command-line tool, handling:
- Command execution and error handling
- Output parsing and formatting
- Parameter validation and defaults

## Usage

` + "```" + `
/{{.Name}} [options]
` + "```" + `

## Implementation

### Command Execution

When invoked, this skill:
1. Validates input parameters
2. Constructs the command with appropriate flags
3. Executes the command using the Bash tool
4. Parses and formats the output
5. Returns results to the user

### Error Handling

The skill handles common error cases:
- Missing required parameters
- Invalid command syntax
- Command execution failures
- Unexpected output formats

## Examples

` + "```" + `
# Basic usage
/{{.Name}}

# With options
/{{.Name}} --option value
` + "```" + `

## Notes

- Requires appropriate permissions for command execution
- May require additional tools to be installed
- See tool documentation for detailed command options
`

// workflowTemplate is the built-in template for workflow skills
const workflowTemplate = `---
name: {{.Name}}
description: {{.Description}}
scope: {{.Scope}}
license: MIT
tools:{{range .Tools}}
  - {{.}}{{end}}
---

# {{.Name}}

A skill that orchestrates multiple steps in a workflow.

## Overview

This skill coordinates multiple operations to accomplish a complex task:
- Sequential step execution
- Conditional branching based on results
- Error recovery and rollback
- Progress tracking and reporting

## Workflow Steps

### Step 1: Preparation

Prepare the environment and validate prerequisites.

` + "```" + `
1. Check for required tools and permissions
2. Validate input parameters
3. Create temporary workspace if needed
` + "```" + `

### Step 2: Execution

Execute the main workflow operations.

` + "```" + `
1. Perform primary operation
2. Collect results
3. Validate output
` + "```" + `

### Step 3: Finalization

Clean up and present results.

` + "```" + `
1. Clean up temporary resources
2. Format and present results
3. Log completion status
` + "```" + `

## Usage

` + "```" + `
/{{.Name}} [options]
` + "```" + `

## Error Handling

The workflow includes error handling at each step:
- Validation errors: Stop before execution
- Execution errors: Attempt recovery or rollback
- Finalization errors: Report but don't fail overall workflow

## Examples

` + "```" + `
# Run complete workflow
/{{.Name}}

# Run with custom options
/{{.Name}} --step 2 --verbose
` + "```" + `

## Dependencies

This workflow may depend on other skills:
{{range .References}}
- {{.}}{{end}}
`

// utilityTemplate is the built-in template for utility skills
const utilityTemplate = `---
name: {{.Name}}
description: {{.Description}}
scope: {{.Scope}}
license: MIT
tools:{{range .Tools}}
  - {{.}}{{end}}
---

# {{.Name}}

A utility skill that provides helper functionality.

## Overview

This skill provides utility functions for:
- Data transformation and formatting
- Common operations and calculations
- Helper functions for other skills

## Features

### Data Processing

Transform and process data in various formats:
- Parse structured data (JSON, YAML, etc.)
- Format output for display or further processing
- Validate data against schemas or rules

### Integration

Integrate with other tools and skills:
- Provide reusable functions
- Share common patterns
- Simplify complex operations

## Usage

` + "```" + `
/{{.Name}} <input> [options]
` + "```" + `

### Options

- ` + "`" + `--format` + "`" + `: Output format (json, yaml, text)
- ` + "`" + `--verbose` + "`" + `: Enable detailed output
- ` + "`" + `--validate` + "`" + `: Validate input before processing

## Examples

` + "```" + `
# Basic usage
/{{.Name}} "input data"

# With formatting
/{{.Name}} "input data" --format json

# With validation
/{{.Name}} "input data" --validate
` + "```" + `

## Implementation Notes

This utility is designed to be:
- **Reusable**: Can be called by other skills
- **Composable**: Works well with other utilities
- **Efficient**: Minimal overhead and fast execution
- **Reliable**: Comprehensive error handling

## API

When called by other skills, this utility provides:

` + "```" + `
Input: <describe expected input format>
Output: <describe output format>
Errors: <describe error conditions>
` + "```" + `
`
