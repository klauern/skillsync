# YNAB Transaction Workflows

Detailed step-by-step workflows for common YNAB transaction management tasks.

## Workflow 1: Daily Transaction Review

**Goal**: Review and categorize yesterday's transactions

```
1. Get uncategorized transactions
   → get_transactions_needing_attention(budget_id, "uncategorized", days_back=1)

2. For each transaction:
   a. Identify likely category based on payee name
   b. Suggest category to user
   c. If approved, categorize_transaction(budget_id, txn_id, category_id)
   d. Move to next transaction

3. Summary
   → Show total categorized, remaining uncategorized
```

**Natural language trigger**: "Review yesterday's transactions"

## Workflow 2: Weekly Budget Cleanup

**Goal**: Clean up all unapproved and uncategorized transactions

```
1. Get all transactions needing attention
   → get_transactions_needing_attention(budget_id, "both", days_back=7)

2. Group by issue type:
   - Uncategorized only
   - Unapproved only
   - Both uncategorized and unapproved

3. Handle uncategorized first:
   → For each: categorize_transaction(...)

4. Handle unapproved:
   → Explain: approval happens in YNAB app
   → List transactions that need manual approval

5. Summary report:
   - Transactions categorized: X
   - Transactions still needing approval: Y
   - All cleaned up: Z
```

**Natural language trigger**: "Clean up my budget" or "Weekly budget review"

## Workflow 3: Bulk Categorization by Payee

**Goal**: Categorize all transactions from a specific payee

```
1. Get recent transactions
   → get_transactions(budget_id, account_id)

2. Filter by payee name (case-insensitive)
   → transactions.filter(payee_name.lower() == target_payee.lower())

3. Show matching transactions to user

4. Ask for category once

5. Batch categorize:
   → For each matching transaction:
      categorize_transaction(budget_id, txn_id, category_id)

6. Confirm all categorized
```

**Natural language trigger**: "Categorize all my Starbucks purchases as Dining Out"

## Workflow 4: Monthly Spending Report

**Goal**: Analyze spending by category for the current month

```
1. Get all transactions for current month
   → since_date = first day of month
   → get_transactions(budget_id, account_id, since_date)

2. Get categories for reference
   → get_categories(budget_id)

3. Group transactions by category_name

4. Calculate totals:
   - Sum positive amounts (expenses) per category
   - Sum negative amounts (income) per category
   - Calculate net spending

5. Sort by total (highest first)

6. Present as formatted table:
   | Category | Transactions | Total |
   |----------|--------------|-------|
   | Groceries | 23 | $542.30 |
   | ...

7. Summary:
   - Total spent: $X
   - Top 3 categories
   - Comparison to budget (if available)
```

**Natural language trigger**: "Show me my spending this month" or "Monthly spending report"

## Workflow 5: Add Split Transaction

**Goal**: Create a single transaction with multiple categories (requires manual split in YNAB)

```
1. Create the main transaction
   → create_transaction(account_id, total_amount, payee_name)

2. Explain to user:
   "I've created the transaction for $X at {payee}.
    To split this across categories, you'll need to:
    1. Open YNAB app/web
    2. Find this transaction (ID: {txn_id})
    3. Click 'Split'
    4. Assign amounts to each category"

3. Optionally offer to list suggested categories based on payee
```

**Natural language trigger**: "Add a split transaction" or "Create transaction for groceries and household items"

## Workflow 6: Reconcile Account

**Goal**: Check account balance against bank

```
1. Get account balance
   → get_account_balance(account_id)

2. Ask user for bank balance

3. Compare:
   - YNAB balance: $X
   - Bank balance: $Y
   - Difference: $Z

4. If difference exists:
   → Get recent unapproved transactions
   → Suggest these might be the cause
   → Offer to show transactions needing attention

5. Guide user:
   "To reconcile in YNAB:
    1. Go to account
    2. Click 'Reconcile'
    3. Enter cleared balance: $Y
    4. Approve matching transactions"
```

**Natural language trigger**: "Reconcile my checking account" or "Check account balance"

## Workflow 7: Find Duplicate Transactions

**Goal**: Identify potential duplicate transactions

```
1. Get recent transactions (30 days)
   → get_transactions(budget_id, account_id, days_back=30)

2. Group by (date, amount, payee_name)

3. Find groups with count > 1

4. Present potential duplicates:
   | Date | Payee | Amount | ID |
   |------|-------|--------|-----|
   | 2025-01-15 | Starbucks | $5.50 | txn-1 |
   | 2025-01-15 | Starbucks | $5.50 | txn-2 |

5. Warning:
   "These look like duplicates. Review in YNAB before deleting.
    Deleting transactions requires YNAB app/web interface."
```

**Natural language trigger**: "Find duplicate transactions" or "Check for duplicates"

## Workflow 8: Quick Cash Transaction

**Goal**: Add cash transaction with minimal friction

```
1. Ask for amount only: "How much?"

2. Use smart defaults:
   - Account: First cash account or checking
   - Category: Suggest based on recent patterns or ask
   - Payee: "Cash" or ask
   - Date: Today

3. Create transaction
   → create_transaction(account_id, amount, payee, category)

4. One-line confirmation
   → "Added $X cash transaction"
```

**Natural language trigger**: "Log a cash purchase" or "Add cash transaction"

## Workflow 9: Review Specific Payee

**Goal**: See all transactions for a specific payee

```
1. Get recent transactions (or custom date range)
   → get_transactions(budget_id, account_id, days_back=90)

2. Filter by payee (fuzzy match):
   - Exact match
   - Case-insensitive
   - Partial match if needed

3. Group by category

4. Present analysis:
   - Total spent at {payee}: $X
   - Number of transactions: Y
   - Average per transaction: $Z
   - Category breakdown:
     * Dining Out: $A (n transactions)
     * Groceries: $B (m transactions)

5. Trends:
   - Frequency (weekly/monthly average)
   - Spending pattern
```

**Natural language trigger**: "Show me all Amazon purchases" or "How much do I spend at Starbucks?"

## Workflow 10: Budget vs Actual

**Goal**: Compare budgeted amounts to actual spending

```
1. Get categories with budget info
   → get_categories(budget_id)

2. Get current month transactions
   → get_transactions(budget_id, account_id)

3. For each category:
   - Budgeted: from categories (in milliunits / 1000)
   - Activity: sum of transactions in category
   - Available: budgeted - activity
   - Progress: (activity / budgeted) * 100%

4. Present as table:
   | Category | Budgeted | Spent | Available | Progress |
   |----------|----------|-------|-----------|----------|
   | Groceries | $500 | $342 | $158 | 68% |

5. Alerts:
   - Over budget (>100%)
   - Close to limit (>90%)
   - Under-utilized (<50% with <5 days left)
```

**Natural language trigger**: "How am I doing against my budget?" or "Budget vs actual"

## Combining Workflows

You can chain workflows together:

**Example**: "Review my budget and create a spending report"
1. Run Workflow 2 (Weekly Budget Cleanup)
2. Run Workflow 4 (Monthly Spending Report)
3. Run Workflow 10 (Budget vs Actual)

## Tips for Custom Workflows

1. **Start simple**: Begin with one goal
2. **Use caching**: Store budget_id and categories
3. **Batch API calls**: Minimize round trips
4. **Present clearly**: Use tables for transaction lists
5. **Confirm actions**: Especially for modifications
6. **Handle errors**: Check for closed accounts, deleted categories
7. **Be conversational**: Hide technical IDs from users when possible