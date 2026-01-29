# Quick Start Guide - YNAB Transactions Skill

Get up and running with the YNAB transactions skill in 5 minutes.

## Prerequisites

1. ✅ YNAB account with active budget
2. ✅ YNAB API key from https://app.ynab.com/settings/developer
3. ✅ MCP server installed (`task install`)
4. ✅ Claude Code CLI installed

## Step 1: Configure API Key

Create or update your `.env` file:

```bash
echo "YNAB_API_KEY=your_api_key_here" > .env
```

Replace `your_api_key_here` with your actual YNAB API key.

## Step 2: Verify MCP Server

Test that the MCP server is configured correctly:

```bash
# Install in development mode with auto-reload
task dev

# Or install for production use
task install
```

## Step 3: First Interaction

Open Claude Code and try your first command:

```
User: Show me my YNAB budgets

Claude: [Uses the skill to list your budgets]
```

If this works, you're all set! 🎉

## Step 4: Set Up Preferences (First Time Only)

Let Claude cache your preferences:

```
User: Set up my YNAB preferences

Claude: I'll help you set up your YNAB preferences...

[Lists your budgets]

Which budget would you like to use as your default?

User: My Budget Name

Claude: [Sets preferred budget and caches categories]

✓ All set! You're ready to go.
```

## Common First Commands

### Check What Needs Attention
```
"Show me my uncategorized transactions"
"What transactions need my attention?"
```

### Review Spending
```
"How much did I spend this month?"
"Show me my grocery spending"
"What did I spend at Starbucks?"
```

### Categorize Transactions
```
"Categorize that transaction as Groceries"
"Put all my Starbucks purchases in Dining Out"
```

### Add Transactions
```
"Add a $15 transaction for lunch"
"Create a transaction: $42 to Amazon in Shopping"
```

### Check Balances
```
"What's my checking account balance?"
"Show all account balances"
```

## Troubleshooting

### Issue: "YNAB_API_KEY not found"

**Solution**: Make sure `.env` file exists with your API key:
```bash
cat .env
# Should show: YNAB_API_KEY=your_key_here
```

### Issue: "No budgets found"

**Solution**:
1. Verify your API key is correct
2. Check you have at least one budget in YNAB
3. Try accessing https://app.ynab.com/ to confirm your account is active

### Issue: Skill doesn't activate

**Solution**:
1. Make sure you're in the project directory
2. Verify `.claude/skills/ynab-transactions/SKILL.md` exists
3. Try being more specific: "Use YNAB skill to show my transactions"

### Issue: "Budget not found" or "Account not found"

**Solution**:
1. Clear cached preferences:
   ```bash
   rm -rf ~/.config/mcp-ynab/
   ```
2. Restart and set preferences again

## Next Steps

Once you're comfortable with basic commands:

1. **Explore workflows**: See [WORKFLOWS.md](WORKFLOWS.md) for advanced workflows
2. **Daily routine**: Set up a daily transaction review habit
3. **Weekly cleanup**: Use the weekly budget cleanup workflow
4. **Monthly analysis**: Generate monthly spending reports

## Learning Path

### Week 1: Basic Commands
- Review uncategorized transactions daily
- Practice categorizing transactions
- Check account balances

### Week 2: Spending Analysis
- Analyze monthly spending
- Compare budget vs actual
- Identify spending patterns

### Week 3: Workflow Automation
- Set up daily review workflow
- Weekly budget cleanup
- Monthly comprehensive review

### Week 4: Advanced Features
- Bulk categorization
- Multi-account management
- Custom analysis requests

## Tips for Success

1. **Start small**: Begin with simple commands and build up
2. **Be specific**: Include time ranges like "this month" or "yesterday"
3. **Let Claude guide**: Claude will ask clarifying questions when needed
4. **Review before confirming**: Claude shows what will change before applying
5. **Use caching**: Budget and category preferences are cached for speed

## Example First Session

```
User: I'm new to the YNAB skill, help me get started

Claude: Welcome! I'll help you set up and use the YNAB skill. Let me start
by checking your budgets...

[Shows budgets]

Which budget would you like to use?

User: Personal Budget

Claude: Great! I'm caching your preferences...

✓ Preferences set for "Personal Budget"
✓ Cached 42 categories

You're all set! Here are some things you can try:

1. "Show me my uncategorized transactions" - See what needs attention
2. "How much did I spend this month?" - Analyze your spending
3. "What's my checking balance?" - Check account balances

What would you like to do first?

User: Show me my uncategorized transactions

Claude: [Shows uncategorized transactions in a table]

I found 8 uncategorized transactions totaling $234.56. Would you like me
to suggest categories for each one?

User: Yes

Claude: [Suggests and applies categories]

✓ All done! Your budget is now clean.

Is there anything else you'd like help with?
```

## Resources

- [Full Documentation](README.md) - Complete guide
- [Examples](EXAMPLES.md) - Real-world usage examples
- [Workflows](WORKFLOWS.md) - Detailed workflow guides
- [Troubleshooting](TROUBLESHOOTING.md) - Solutions to common issues

## Getting Help

If you run into issues:

1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
2. Verify your YNAB API key is valid
3. Make sure MCP server is running (`task dev`)
4. Check YNAB service status: https://status.youneedabudget.com/

## Quick Reference Card

**Transaction Management**
- Review: "Show uncategorized transactions"
- Categorize: "Categorize txn-123 as Groceries"
- Create: "Add $15 transaction for coffee"

**Analysis**
- Monthly: "Show spending this month"
- Category: "How much on groceries?"
- Payee: "All Amazon purchases"

**Accounts**
- Balance: "Checking account balance"
- All: "Show all balances"
- Reconcile: "Reconcile checking account"

**Workflows**
- Daily: "Review yesterday's transactions"
- Weekly: "Clean up my budget"
- Monthly: "Complete monthly review"

---

**Ready to start?** Try: `"Show me my YNAB budgets"`