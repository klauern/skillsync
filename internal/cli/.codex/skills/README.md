# YNAB Transactions Skill

A Claude Code skill for managing YNAB (You Need A Budget) transactions through natural language.

## Overview

This skill enables Claude to help you manage your YNAB budget by:
- Reviewing uncategorized and unapproved transactions
- Categorizing transactions individually or in bulk
- Creating new transactions
- Analyzing spending patterns
- Checking account balances
- Managing budget preferences

## Quick Start

Once this skill is active, you can use natural language commands like:

```
"Show me my uncategorized transactions"
"Categorize that Starbucks purchase as Dining Out"
"How much did I spend on groceries this month?"
"Add a $42 transaction to Amazon in Shopping"
"What's my checking account balance?"
```

Claude will automatically:
1. Use your cached budget ID (or help you select one)
2. Retrieve and cache categories for faster operations
3. Present data in clear, formatted tables
4. Guide you through multi-step workflows
5. Confirm all modifications

## Files in This Skill

- **SKILL.md** - Main skill definition with workflows and best practices
- **WORKFLOWS.md** - Detailed step-by-step workflows for common tasks
- **TROUBLESHOOTING.md** - Solutions to common issues and errors
- **README.md** - This file

## Requirements

This skill requires the **mcp-ynab** MCP server to be running and configured with a valid YNAB API key.

### Setup

1. Install and configure the MCP server:
   ```bash
   cd /path/to/mcp-ynab
   task install
   ```

2. Set your YNAB API key:
   ```bash
   # In .env file
   YNAB_API_KEY=your_api_key_here
   ```

3. Get your API key from: https://app.ynab.com/settings/developer

4. The skill will be automatically available when you use Claude Code in this project

## Common Workflows

### Daily Transaction Review
Review and categorize yesterday's or today's transactions:
```
"Review yesterday's transactions"
"What transactions need categorizing?"
```

### Weekly Budget Cleanup
Clean up all uncategorized and unapproved transactions:
```
"Clean up my budget"
"Weekly budget review"
```

### Spending Analysis
Analyze your spending patterns:
```
"Show me my spending this month"
"How much do I spend at Starbucks?"
"Budget vs actual comparison"
```

### Quick Transaction Entry
Add transactions quickly:
```
"Add a $15 transaction for lunch"
"Log a cash purchase of $5.50"
```

### Account Management
Check balances and reconcile:
```
"What's my checking balance?"
"Show all account balances"
"Reconcile my credit card"
```

## Tips for Best Results

1. **Be specific about time ranges**: "this month", "yesterday", "last 30 days"
2. **Use natural language**: Claude understands context and can infer details
3. **Let Claude guide you**: For complex tasks, Claude will ask clarifying questions
4. **Review before confirming**: Claude will show you what will change before making modifications
5. **Use cached data**: Budget and category preferences are cached for speed

## How It Works

The skill uses a two-layer architecture:

1. **MCP Server Layer** (mcp-ynab)
   - Handles YNAB API authentication
   - Provides low-level tools for API operations
   - Manages data caching and configuration

2. **Skill Layer** (this skill)
   - Provides high-level workflow guidance
   - Teaches Claude when and how to use MCP tools
   - Handles natural language understanding
   - Orchestrates multi-step operations

## Example Session

```
User: Help me categorize my recent transactions

Claude: I'll help you categorize your transactions. First, let me get your
uncategorized transactions...

[Calls get_transactions_needing_attention]

I found 5 uncategorized transactions:

| ID | Date | Payee | Amount |
|----|------|-------|--------|
| txn-1 | 2025-12-12 | Starbucks | $5.50 |
| txn-2 | 2025-12-12 | Safeway | $87.23 |
| txn-3 | 2025-12-11 | Shell | $45.00 |
| txn-4 | 2025-12-11 | Netflix | $15.99 |
| txn-5 | 2025-12-10 | Amazon | $42.37 |

Would you like me to:
1. Suggest categories for each one?
2. Let you categorize them manually?
3. Show me your category list first?

User: Suggest categories

Claude: Based on the payee names, here are my suggestions:

1. Starbucks → Dining Out
2. Safeway → Groceries
3. Shell → Gas & Fuel
4. Netflix → Subscriptions
5. Amazon → Shopping

Should I apply these categorizations? (yes/no/modify)

User: yes

Claude: I'll categorize all 5 transactions now...

[Categorizes each transaction]

✓ All 5 transactions categorized successfully!

Summary:
- Dining Out: $5.50 (1 transaction)
- Groceries: $87.23 (1 transaction)
- Gas & Fuel: $45.00 (1 transaction)
- Subscriptions: $15.99 (1 transaction)
- Shopping: $42.37 (1 transaction)

Total categorized: $196.09

Is there anything else you'd like help with?
```

## Advanced Usage

### Custom Workflows

You can create custom workflows by chaining commands:

```
"Review my budget, create a spending report, and check my balances"
```

Claude will:
1. Run the weekly cleanup workflow
2. Generate a monthly spending report
3. Check all account balances
4. Present a comprehensive summary

### Batch Operations

For bulk operations:

```
"Categorize all my Starbucks purchases as Dining Out"
```

Claude will:
1. Find all Starbucks transactions
2. Show you what will be categorized
3. Ask for confirmation
4. Categorize them all at once
5. Report results

### Integration with Other Tools

This skill can work alongside other Claude Code skills:

```
"Review my YNAB transactions and create a spending summary spreadsheet"
```

Claude can:
1. Use the ynab-transactions skill to get data
2. Use a spreadsheet skill to create the summary
3. Present both outputs

## Limitations

1. **Transaction Approval**: Must be done in YNAB app (API limitation)
2. **Split Transactions**: Must be created/edited in YNAB app
3. **Budget Creation**: Must be done in YNAB app
4. **Account Management**: Opening/closing accounts requires YNAB app
5. **Rate Limits**: YNAB API allows 200 requests/hour

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed solutions to common issues.

Quick fixes:
- **API key errors**: Check `.env` file has valid YNAB_API_KEY
- **Budget not found**: Run `get_budgets()` and set preferred budget
- **Outdated categories**: Clear cache and run `cache_categories()`
- **Slow performance**: Use date ranges and caching

## Contributing

To improve this skill:

1. Add new workflows to WORKFLOWS.md
2. Document common issues in TROUBLESHOOTING.md
3. Update examples in SKILL.md
4. Test with real YNAB data

## Resources

- [YNAB API Documentation](https://api.ynab.com/)
- [YNAB Developer Settings](https://app.ynab.com/settings/developer)
- [MCP Server Repository](../../README.md)
- [Claude Code Skills Documentation](https://docs.claude.com/docs/agents-and-tools/agent-skills)

## License

This skill is part of the mcp-ynab project and uses the same license as the parent project.