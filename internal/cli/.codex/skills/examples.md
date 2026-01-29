# YNAB Transactions Skill - Usage Examples

Real-world examples of using the YNAB transactions skill with Claude Code.

## Example 1: Getting Started

**First Time Setup**

```
User: I want to start using the YNAB skill

Claude: I'll help you get started with YNAB! First, let me check your budgets...

[Calls get_budgets()]

I found these budgets:
- Personal Budget (ID: budget-abc123)
- Business Budget (ID: budget-xyz789)

Which budget would you like to use as your default?

User: Personal Budget

Claude: Perfect! I'll set "Personal Budget" as your preferred budget and cache
your categories for faster operations.

[Calls set_preferred_budget_id("budget-abc123")]
[Calls get_categories("budget-abc123")]
[Calls cache_categories("budget-abc123")]

✓ All set! I've cached 42 categories from your Personal Budget.

You can now:
- "Show me uncategorized transactions"
- "What's my spending this month?"
- "Add a new transaction"

What would you like to do?
```

## Example 2: Daily Transaction Review

**Morning Routine: Review Yesterday's Transactions**

```
User: Review yesterday's transactions

Claude: I'll show you yesterday's uncategorized transactions...

[Calls get_transactions_needing_attention(budget_id, "uncategorized", days_back=1)]

Found 3 transactions from yesterday that need categorizing:

| ID | Date | Payee | Amount | Account |
|----|------|-------|--------|---------|
| txn-001 | 2025-12-12 | Corner Bakery | $12.45 | Checking |
| txn-002 | 2025-12-12 | Shell Gas | $48.00 | Credit Card |
| txn-003 | 2025-12-12 | Target | $67.89 | Checking |

Let me suggest categories:

1. Corner Bakery → Dining Out
2. Shell Gas → Gas & Fuel
3. Target → Shopping (or would you like to split this?)

Should I apply these suggestions?

User: Yes, but Target should be Groceries

Claude: Got it! I'll categorize them as:
1. Corner Bakery → Dining Out
2. Shell Gas → Gas & Fuel
3. Target → Groceries

[Categorizes each transaction]

✓ Done! All 3 transactions categorized.

Summary:
- Dining Out: $12.45
- Gas & Fuel: $48.00
- Groceries: $67.89
Total: $128.34

You're all caught up! 🎉
```

## Example 3: Quick Transaction Entry

**Adding a Cash Purchase**

```
User: I just bought coffee for $5.50 cash

Claude: I'll add that transaction for you.

[Calls get_accounts(budget_id) to find cash/checking account]

Which account?
- Checking (ID: acc-123)
- Cash (ID: acc-456)
- Savings (ID: acc-789)

User: Cash

Claude: And which category?
- Dining Out
- Groceries
- Coffee (if you have a specific category)
- Other

User: Dining Out

Claude: Creating transaction...

[Calls create_transaction(
  account_id="acc-456",
  amount=5.50,
  payee_name="Coffee",
  category_name="Dining Out",
  memo="Cash purchase"
)]

✓ Added $5.50 cash transaction to "Coffee" in Dining Out category.

Your Cash account balance is now: $142.50
```

## Example 4: Bulk Categorization

**Categorizing Multiple Transactions from Same Payee**

```
User: Categorize all my Starbucks purchases as Dining Out

Claude: Let me find all your Starbucks transactions...

[Calls get_transactions(budget_id, account_id, days_back=90)]
[Filters by payee_name containing "Starbucks"]

Found 12 Starbucks transactions in the last 90 days:

| Date | Amount | Status |
|------|--------|--------|
| 2025-12-12 | $5.50 | Uncategorized |
| 2025-12-10 | $6.25 | Uncategorized |
| 2025-12-08 | $5.50 | Uncategorized |
| 2025-12-05 | $7.00 | Uncategorized |
| ... | ... | ... |

Total: $72.50 across 12 transactions

Should I categorize all 12 as "Dining Out"?

User: Yes

Claude: Categorizing all 12 Starbucks transactions...

[Calls categorize_transaction for each transaction]

✓ Successfully categorized 12 transactions as Dining Out
Total amount: $72.50

Would you like me to show your Dining Out spending summary?
```

## Example 5: Monthly Spending Analysis

**Understanding Spending Patterns**

```
User: How much did I spend on groceries this month?

Claude: Let me analyze your grocery spending for December 2025...

[Calls get_transactions(budget_id, account_id)]
[Filters by category_name == "Groceries" and current month]

## Grocery Spending - December 2025

Total Spent: $542.30
Number of Transactions: 18
Average per Transaction: $30.13

### Top Stores:
1. Safeway: $245.67 (9 transactions)
2. Trader Joe's: $156.23 (5 transactions)
3. Whole Foods: $89.50 (2 transactions)
4. Farmers Market: $50.90 (2 transactions)

### Weekly Breakdown:
- Week 1: $145.20
- Week 2: $198.45
- Week 3 (current): $198.65

### Budget Status:
- Budgeted: $600.00
- Spent: $542.30
- Remaining: $57.70
- Progress: 90%

You're on track! You have $57.70 left for the rest of the month.
```

## Example 6: Finding and Categorizing Unapproved

**Weekly Budget Cleanup**

```
User: Clean up my budget

Claude: I'll help you clean up your budget. Let me check what needs attention...

[Calls get_transactions_needing_attention(budget_id, "both", days_back=7)]

Found transactions needing attention:

## Uncategorized (8 transactions)
| ID | Date | Payee | Amount |
|----|------|-------|--------|
| txn-101 | 2025-12-12 | Amazon | $42.99 |
| txn-102 | 2025-12-11 | Spotify | $9.99 |
| ... | ... | ... | ... |

## Unapproved (15 transactions)
| ID | Date | Payee | Amount | Category |
|----|------|-------|--------|----------|
| txn-201 | 2025-12-10 | Target | $67.89 | Groceries |
| txn-202 | 2025-12-09 | Shell | $45.00 | Gas & Fuel |
| ... | ... | ... | ... | ... |

Let's handle the uncategorized ones first. Would you like me to:
1. Suggest categories for each one
2. Show your category list so you can decide
3. Skip to the unapproved transactions

User: Suggest categories

Claude: Based on payee names and your spending patterns:

1. Amazon ($42.99) → Shopping
2. Spotify ($9.99) → Subscriptions
3. Netflix ($15.99) → Subscriptions
4. Safeway ($87.23) → Groceries
5. Chevron ($52.00) → Gas & Fuel
6. Uber ($18.50) → Transportation
7. CVS ($24.35) → Health & Medicine
8. Starbucks ($6.75) → Dining Out

Apply all suggestions? (yes/no/modify)

User: yes

Claude: Categorizing...

[Categorizes all 8 transactions]

✓ Categorized 8 transactions ($306.80 total)

For the 15 unapproved transactions: These need to be approved in the YNAB app.
They're already categorized correctly.

To approve them:
1. Open YNAB app or web
2. Go to your budget
3. Select unapproved transactions
4. Press 'A' or click 'Approve'

Your budget is now clean! All transactions are categorized. ✨
```

## Example 7: Budget vs Actual Analysis

**Understanding Budget Performance**

```
User: How am I doing against my budget?

Claude: I'll analyze your budget performance for December 2025...

[Calls get_categories(budget_id)]
[Calls get_transactions(budget_id, account_id)]
[Analyzes budget vs actual spending]

## Budget vs Actual - December 2025

### Overview
- Total Budgeted: $3,500.00
- Total Spent: $2,847.32
- Remaining: $652.68
- Days Left: 8
- Daily Budget Available: $81.59

### Category Breakdown

| Category | Budgeted | Spent | Available | Progress |
|----------|----------|-------|-----------|----------|
| Groceries | $600 | $542 | $58 | 90% ● |
| Dining Out | $300 | $287 | $13 | 96% ● |
| Gas & Fuel | $200 | $156 | $44 | 78% ● |
| Subscriptions | $150 | $149 | $1 | 99% ⚠️ |
| Shopping | $400 | $523 | -$123 | 131% 🔴 |
| Entertainment | $250 | $178 | $72 | 71% ● |
| Health | $200 | $124 | $76 | 62% ● |
| Transportation | $300 | $267 | $33 | 89% ● |
| Utilities | $450 | $450 | $0 | 100% ⚠️ |
| Other | $650 | $371 | $279 | 57% ● |

### Alerts
🔴 **Over Budget**: Shopping ($123 over)
⚠️ **Nearly Depleted**: Subscriptions ($1 left), Utilities ($0 left)
✓ **On Track**: Groceries, Gas, Entertainment, Health, Transportation

### Recommendations
1. Watch Shopping category - you're $123 over budget
2. Subscriptions is nearly depleted - any more charges will go over
3. Good progress on Groceries, Gas, and Entertainment
4. Consider moving funds from "Other" to cover Shopping overage

Would you like to see details for any specific category?
```

## Example 8: Account Balance Check

**Quick Balance Check**

```
User: What's my checking account balance?

Claude: Let me check your checking account...

[Calls get_accounts(budget_id)]
[Calls get_account_balance(account_id)]

Your checking account balance is: **$2,547.89**

Recent activity (last 7 days):
- Starting balance: $2,892.45
- Total inflows: $0.00
- Total outflows: -$344.56
- Current balance: $2,547.89

Pending transactions (unapproved): 3 transactions totaling $127.45
Working balance: $2,420.44

Would you like to see:
1. Recent transactions
2. All account balances
3. Reconcile this account
```

## Example 9: Multi-Account Overview

**Check All Balances**

```
User: Show all account balances

Claude: Here's your complete account overview...

[Calls get_accounts(budget_id)]
[Calls get_account_balance for each account]

## Account Summary

### Checking Accounts
- Main Checking: $2,547.89
- Joint Checking: $1,234.56
**Total Checking: $3,782.45**

### Savings Accounts
- Emergency Fund: $10,500.00
- Vacation Fund: $2,300.00
**Total Savings: $12,800.00**

### Credit Cards
- Chase Sapphire: -$1,247.32 (statement: $2,500 limit)
- Amex Blue: -$567.89 (statement: $15,000 limit)
**Total Credit Card Debt: -$1,815.21**

### Assets Summary
- Total Cash: $16,582.45
- Total Debt: -$1,815.21
- **Net Worth: $14,767.24**

All accounts are up to date. Last sync: 2 hours ago
```

## Example 10: Finding Duplicate Transactions

**Checking for Duplicates**

```
User: Check for duplicate transactions

Claude: Scanning for potential duplicate transactions...

[Calls get_transactions(budget_id, account_id, days_back=30)]
[Groups by (date, amount, payee_name)]
[Finds groups with count > 1]

Found 2 potential duplicate groups:

### Group 1: Starbucks on 2025-12-10
| ID | Date | Payee | Amount | Category | Status |
|----|------|-------|--------|----------|--------|
| txn-456 | 2025-12-10 | Starbucks | $5.50 | Dining Out | Approved |
| txn-457 | 2025-12-10 | Starbucks | $5.50 | Uncategorized | Unapproved |

### Group 2: Amazon on 2025-12-08
| ID | Date | Payee | Amount | Category | Status |
|----|------|-------|--------|----------|--------|
| txn-234 | 2025-12-08 | Amazon | $42.99 | Shopping | Approved |
| txn-235 | 2025-12-08 | Amazon | $42.99 | Shopping | Approved |

⚠️ These look like duplicates, but review carefully:
- Different transactions with the same amount/date are possible
- Check memo fields in YNAB app for details
- To delete duplicates, use the YNAB app or web interface

Would you like me to show the memo fields for any of these?
```

## Tips for Effective Use

1. **Be conversational**: Use natural language, Claude understands context
2. **Specify time ranges**: "this month", "yesterday", "last week"
3. **Let Claude guide**: For complex tasks, Claude will ask clarifying questions
4. **Review before confirming**: Claude shows what will change
5. **Chain commands**: "Clean up my budget and show me spending analysis"

## Advanced Combinations

**Example: Complete Monthly Review**

```
User: Do a complete monthly review

Claude: I'll perform a comprehensive monthly review of your budget...

[Runs multiple workflows in sequence]

# Monthly Budget Review - December 2025

## 1. Transaction Cleanup
✓ Categorized 12 uncategorized transactions
⚠️ 8 transactions still need approval in YNAB app

## 2. Spending Analysis
- Total spent: $2,847.32
- Average daily spend: $91.85
- Top category: Groceries ($542.30)

## 3. Budget Performance
- On track: 7 categories
- Over budget: 1 category (Shopping)
- Under-utilized: 2 categories

## 4. Account Health
- Total cash: $16,582.45
- Total debt: -$1,815.21
- Net worth: $14,767.24
- Change from last month: +$1,234.56 (9.1% increase)

## 5. Recommendations
1. Review Shopping category - $123 over budget
2. Consider increasing Groceries budget (consistently at 90%+)
3. Good progress on debt reduction
4. Emergency fund is healthy at $10,500

Would you like details on any section?
```

These examples demonstrate the full range of the YNAB transactions skill. Start with simple commands and work up to more complex workflows as you get comfortable!