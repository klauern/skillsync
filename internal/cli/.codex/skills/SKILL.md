# YNAB Transaction Management

This Skill helps you efficiently manage your YNAB (You Need A Budget) transactions through natural language commands. It leverages the mcp-ynab server to interact with your budget.

## Common Workflows

### 1. Review Transactions Needing Attention

**When to use**: User wants to categorize transactions, approve transactions, or clean up their budget.

**Workflow**:
1. First, ensure we have a budget ID. If not cached, call `get_budgets()` and ask the user which budget to use
2. Call `set_preferred_budget_id(budget_id)` to cache it for future use
3. Call `get_categories(budget_id)` to see available categories
4. Call `cache_categories(budget_id)` to cache them for future use
5. Call `get_transactions_needing_attention(budget_id, filter_type="both")` to see what needs work
6. Present the results and offer to help categorize transactions

**Example user requests**:
- "Show me my uncategorized transactions"
- "What transactions need my attention?"
- "Help me clean up my budget"

### 2. Categorize Transactions

**When to use**: User wants to assign categories to one or more transactions.

**Workflow**:
1. Get the transaction ID from previous output or ask the user
2. Get the category ID from cached categories or ask the user
3. Call `categorize_transaction(budget_id, transaction_id, category_id)`
4. Confirm the categorization was successful

**Example user requests**:
- "Categorize transaction beads-abc as Groceries"
- "Put that coffee purchase in Dining Out"
- "Mark all these as Gas & Fuel"

### 3. Create New Transaction

**When to use**: User wants to manually add a transaction to YNAB.

**Workflow**:
1. Ensure we have budget_id and account_id (get accounts if needed)
2. Gather: amount (in dollars), payee_name, optional category_name, optional memo
3. Call `create_transaction(account_id, amount, payee_name, category_name, memo)`
4. Confirm creation and show the transaction details

**Example user requests**:
- "Add a $5.50 transaction for coffee at Starbucks"
- "Create a transaction: $42 to Amazon in the Shopping category"
- "Log a cash purchase"

### 4. Analyze Spending

**When to use**: User wants to understand their spending patterns.

**Workflow**:
1. Get accounts and transactions for the relevant period
2. Call `get_transactions(budget_id, account_id)` for recent transactions
3. Analyze and summarize by category, payee, or time period
4. Present insights in an easy-to-understand format

**Example user requests**:
- "How much did I spend on dining out this month?"
- "Show me all my Amazon purchases"
- "What's my spending pattern?"

### 5. Account Balance Check

**When to use**: User wants to know their current account balance.

**Workflow**:
1. Get account_id if not provided
2. Call `get_account_balance(account_id)`
3. Present the balance clearly

**Example user requests**:
- "What's my checking account balance?"
- "How much is in my savings?"
- "Show all account balances"

## Best Practices

### Starting Fresh
Always check if we have a preferred budget ID cached:
1. Try to use cached budget_id from preferences
2. If none exists, call `get_budgets()` and let user select
3. Cache the selection with `set_preferred_budget_id(budget_id)`

### Category Management
Cache categories once per session to avoid repeated API calls:
1. Call `get_categories(budget_id)` to see all categories
2. Call `cache_categories(budget_id)` to cache them
3. Reference cached categories when categorizing

### Transaction IDs
YNAB uses different types of transaction IDs:
- **id**: Direct transaction ID (most common, shown in tables)
- **import_id**: YNAB import format (YNAB:[amount]:[date]:[occurrence])
- **transfer_transaction_id**: For transfer transactions
- **matched_transaction_id**: For matched transactions

Always use the `id` from the transaction table unless the user specifically provides an import_id.

### Handling Multiple Transactions
When categorizing multiple transactions:
1. Present them in a clear table format
2. Process them one at a time
3. Confirm each categorization
4. Summarize the batch at the end

### Error Handling
If a tool call fails:
1. Check if the budget_id is still valid
2. Verify the account_id exists and is open
3. Ensure category_id is valid (not deleted or hidden)
4. Provide helpful error messages to the user

## Quick Reference

### Filter Types for `get_transactions_needing_attention`
- `"uncategorized"`: Only show transactions without a category
- `"unapproved"`: Only show transactions not yet approved
- `"both"`: Show all transactions needing attention (default)

### Amount Format
All amounts in tool calls are in **dollars** (not YNAB's milliunits):
- User says: "$42.50"
- You call: `create_transaction(..., amount=42.50, ...)`

### Date Handling
- Most tools use current month by default
- Use `days_back` parameter to look further back
- Transactions are sorted by date (most recent first)

## Example Conversations

### Example 1: Clean Up Budget
**User**: "Help me categorize my recent transactions"

**Claude**:
1. Calls `get_budgets()` → shows "Personal Budget"
2. Calls `set_preferred_budget_id("budget-123")`
3. Calls `get_categories("budget-123")` → displays all categories
4. Calls `cache_categories("budget-123")`
5. Calls `get_transactions_needing_attention("budget-123", "both")`
6. Shows table of uncategorized/unapproved transactions
7. Offers to help categorize them one by one or in bulk

### Example 2: Quick Transaction Entry
**User**: "Add a $15 transaction for lunch at Chipotle"

**Claude**:
1. Uses cached budget_id (or gets it if not cached)
2. Calls `get_accounts(budget_id)` → shows checking account
3. Asks: "Which account?" or uses default checking
4. Asks: "Which category?" or offers "Dining Out"
5. Calls `create_transaction(account_id, 15.00, "Chipotle", "Dining Out")`
6. Confirms: "Created $15.00 transaction to Chipotle in Dining Out category"

### Example 3: Spending Analysis
**User**: "How much did I spend on groceries this month?"

**Claude**:
1. Uses cached budget_id
2. Calls `get_categories(budget_id)` → finds "Groceries" category_id
3. Calls `get_transactions(budget_id, account_id)` for each account
4. Filters transactions by category_name == "Groceries"
5. Sums amounts and presents: "You spent $342.50 on groceries this month across 12 transactions"

## Tips for Effective Use

1. **Be conversational**: Users don't need to know about budget_id or account_id - Claude handles the technical details
2. **Batch operations**: When possible, handle multiple related tasks in sequence
3. **Provide context**: Always show what you're doing and why
4. **Confirm actions**: Especially for categorization and transaction creation
5. **Use tables**: Present transaction data in clear markdown tables
6. **Cache intelligently**: Reuse budget_id and categories across the session

## See Also

- [WORKFLOWS.md](WORKFLOWS.md) - Detailed workflow examples
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [MCP Server Documentation](../../README.md) - Technical API reference