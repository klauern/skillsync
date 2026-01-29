# YNAB Transactions Troubleshooting

Common issues and their solutions when working with the YNAB MCP server.

## API Connection Issues

### Problem: "YNAB_API_KEY not found"

**Symptoms**:
```
ValueError: YNAB_API_KEY not found in environment variables
```

**Solutions**:
1. Check environment variable is set:
   ```bash
   echo $YNAB_API_KEY
   ```

2. If not set, add to `.env` file:
   ```bash
   echo "YNAB_API_KEY=your_api_key_here" > .env
   ```

3. Get API key from: https://app.ynab.com/settings/developer

4. Restart the MCP server for changes to take effect

### Problem: "401 Unauthorized"

**Symptoms**:
- API calls fail with 401 error
- "Invalid access token" message

**Solutions**:
1. API key might be expired or revoked
2. Generate new API key from YNAB settings
3. Update `.env` file with new key
4. Restart MCP server

### Problem: "429 Too Many Requests"

**Symptoms**:
- Rate limit exceeded
- Some requests succeed, others fail

**Solutions**:
1. YNAB API has rate limits (200 requests/hour)
2. Use caching to reduce API calls:
   - Cache budget_id with `set_preferred_budget_id`
   - Cache categories with `cache_categories`
3. Batch operations when possible
4. Wait before retrying (check YNAB headers for retry time)

## Budget and Account Issues

### Problem: "Budget not found"

**Symptoms**:
- Can't find expected budget
- `get_budgets()` returns empty or wrong budget

**Solutions**:
1. Verify budget exists in YNAB app
2. Check if budget was deleted or archived
3. Clear cached budget_id:
   ```bash
   rm ~/.config/mcp-ynab/preferred_budget_id.json
   ```
4. Call `get_budgets()` to see all available budgets

### Problem: "Account is closed"

**Symptoms**:
- Account doesn't appear in `get_accounts()` results
- Transactions fail for specific account

**Solutions**:
1. Check account status in YNAB app
2. Closed accounts are filtered out by default
3. Reopen account in YNAB if needed
4. Use a different active account

### Problem: "Account ID mismatch"

**Symptoms**:
- Transaction creation fails with "Account not found"
- Account IDs don't match between calls

**Solutions**:
1. Always use account IDs from `get_accounts()` response
2. Don't hardcode account IDs (they can change)
3. Match by account name instead:
   ```python
   accounts = get_accounts(budget_id)
   checking = [a for a in accounts if a['name'] == 'Checking'][0]
   account_id = checking['id']
   ```

## Transaction Issues

### Problem: "Category not found"

**Symptoms**:
- Categorization fails
- "Invalid category_id" error

**Solutions**:
1. Category might be deleted or hidden
2. Call `get_categories(budget_id)` to see current categories
3. Update cached categories: `cache_categories(budget_id)`
4. Use exact category_id from response
5. Check category isn't in deleted group

### Problem: "Transaction not found by import_id"

**Symptoms**:
- Can't find transaction using import_id
- `categorize_transaction` fails

**Solutions**:
1. Verify import_id format: `YNAB:[milliunit_amount]:[date]:[occurrence]`
2. Use regular transaction `id` instead (from transaction table)
3. Check date range includes the transaction
4. Example correct usage:
   ```python
   # Get from table: ID column
   categorize_transaction(budget_id, "txn-abc123", category_id, id_type="id")

   # Not import_id unless explicitly needed
   ```

### Problem: "Amount in wrong units"

**Symptoms**:
- Transaction shows wrong amount
- $42 becomes $42,000 or $0.042

**Solutions**:
1. MCP tools use **dollars**, not milliunits
2. Correct: `create_transaction(..., amount=42.50, ...)`
3. Wrong: `create_transaction(..., amount=42500, ...)`
4. YNAB API uses milliunits internally (amount * 1000), but MCP handles conversion

### Problem: "Can't approve transactions"

**Symptoms**:
- User wants to approve transactions
- No approve function available

**Solutions**:
1. Transaction approval must be done in YNAB app/web
2. MCP server doesn't have approve functionality (YNAB API limitation)
3. Guide user:
   ```
   "To approve transactions:
    1. Open YNAB app or web interface
    2. Select transactions
    3. Click 'Approve' or press 'A'"
   ```
4. Focus on categorization instead, which IS available via MCP

## Caching Issues

### Problem: "Categories are outdated"

**Symptoms**:
- New categories don't appear
- Deleted categories still show up

**Solutions**:
1. Clear category cache:
   ```bash
   rm ~/.config/mcp-ynab/budget_category_cache.json
   ```
2. Refresh: `cache_categories(budget_id)`
3. Cache expires when YNAB categories change

### Problem: "Wrong budget selected"

**Symptoms**:
- Operations apply to wrong budget
- Can't switch budgets

**Solutions**:
1. Clear preferred budget:
   ```bash
   rm ~/.config/mcp-ynab/preferred_budget_id.json
   ```
2. Set new preferred budget:
   ```python
   set_preferred_budget_id("new_budget_id")
   ```
3. Or explicitly pass budget_id to each call

## Performance Issues

### Problem: "Slow response times"

**Symptoms**:
- Tools take long time to respond
- Timeouts on large operations

**Solutions**:
1. Reduce date range:
   ```python
   # Instead of all transactions
   get_transactions(..., days_back=30)  # Just last 30 days
   ```

2. Use caching:
   - Set preferred budget once
   - Cache categories once per session

3. Batch operations:
   - Get all needed data in one call
   - Process locally instead of multiple API calls

4. Limit results:
   - Filter by account
   - Filter by date range
   - Use specific queries instead of "get everything"

### Problem: "Memory issues with large datasets"

**Symptoms**:
- Crashes when processing many transactions
- Out of memory errors

**Solutions**:
1. Process in chunks:
   ```python
   # Instead of get_transactions for all time
   # Process month by month
   for month in range(12):
       transactions = get_transactions(..., month_range)
       process(transactions)
   ```

2. Use filters:
   - `get_transactions_needing_attention` instead of all transactions
   - Filter by account
   - Limit to recent transactions

## Data Consistency Issues

### Problem: "Balance doesn't match YNAB"

**Symptoms**:
- `get_account_balance` shows different amount than YNAB app

**Solutions**:
1. Check cleared vs uncleared transactions
2. YNAB might show "working balance" (includes pending)
3. MCP shows "cleared balance"
4. Reconcile account in YNAB to resolve
5. Refresh YNAB app/web to sync

### Problem: "Duplicate transactions appear"

**Symptoms**:
- Same transaction shows up multiple times
- Import creates duplicates

**Solutions**:
1. YNAB import_id prevents duplicates
2. If using `create_transaction`, duplicates are possible
3. Check for duplicates before creating:
   ```python
   recent = get_transactions(budget_id, account_id, days_back=7)
   # Check if similar transaction exists
   ```
4. Use YNAB's duplicate detection (works in app, not via API)

## Debugging Tips

### Enable Debug Logging

Add to your MCP server initialization:
```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

### Check MCP Server Logs

Look for errors in server output:
```bash
# If running in dev mode
task dev

# Check for error messages in output
```

### Verify Tool Availability

Check which tools are available:
```python
# In Claude Code, list available tools
# Should see: mcp__ynab__* tools
```

### Test Individual Tools

Test tools in isolation:
```python
# Start with simple read-only tools
result = get_budgets()
print(result)

# Then try account operations
result = get_accounts(budget_id)
print(result)

# Finally test write operations
result = create_transaction(...)
print(result)
```

### Check Configuration Files

Verify config directory and files:
```bash
ls -la ~/.config/mcp-ynab/
cat ~/.config/mcp-ynab/preferred_budget_id.json
cat ~/.config/mcp-ynab/budget_category_cache.json
```

## Getting Help

If issues persist:

1. **Check YNAB API Status**: https://status.youneedabudget.com/
2. **Review YNAB API Docs**: https://api.ynab.com/
3. **Check MCP Server Issues**: GitHub repository issues
4. **Verify Environment**:
   ```bash
   python --version  # Should be 3.12+
   pip list | grep ynab
   echo $YNAB_API_KEY | cut -c1-10  # Verify key exists (first 10 chars)
   ```

## Common Error Messages Reference

| Error | Cause | Solution |
|-------|-------|----------|
| `YNAB_API_KEY not found` | Missing API key | Add to `.env` file |
| `401 Unauthorized` | Invalid/expired API key | Generate new key |
| `404 Not Found` | Invalid budget/account/transaction ID | Verify ID exists |
| `429 Too Many Requests` | Rate limit exceeded | Use caching, reduce calls |
| `ValueError: Invalid amount` | Wrong amount format | Use dollars, not milliunits |
| `Category not found` | Deleted/hidden category | Refresh category cache |
| `Transaction not found` | Wrong transaction ID | Use `id` not `import_id` |

## Still Stuck?

If you're still experiencing issues:
1. Restart the MCP server
2. Clear all cache files
3. Verify YNAB API key is valid
4. Test with YNAB web interface to confirm expected behavior
5. Check for YNAB service outages