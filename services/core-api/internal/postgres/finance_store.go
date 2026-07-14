package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/finance"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func normalizeCurrency(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return "", platform.Invalid("invalid_currency", "currency must be a three-letter code")
	}
	return value, nil
}

func accountCode(parts ...string) string {
	for index := range parts {
		parts[index] = strings.ToLower(strings.TrimSpace(parts[index]))
		parts[index] = strings.NewReplacer(" ", "_", "/", "_", ":", "_").Replace(parts[index])
	}
	return strings.Join(parts, ":")
}

func ensureAccount(ctx context.Context, tx pgx.Tx, tenantID, walletID, code, name, accountType, purpose, currency string) (finance.Account, error) {
	var item finance.Account
	err := tx.QueryRow(ctx, `
		INSERT INTO ledger_accounts (tenant_id,wallet_id,code,name,account_type,purpose,currency)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id,code) DO NOTHING
		RETURNING id,tenant_id,COALESCE(wallet_id::text,''),code,name,account_type,purpose,currency`,
		tenantID, nullableUUID(walletID), code, name, accountType, purpose, currency,
	).Scan(&item.ID, &item.TenantID, &item.WalletID, &item.Code, &item.Name, &item.AccountType, &item.Purpose, &item.Currency)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `SELECT id,tenant_id,COALESCE(wallet_id::text,''),code,name,account_type,purpose,currency FROM ledger_accounts WHERE tenant_id=$1 AND code=$2`, tenantID, code).Scan(
			&item.ID, &item.TenantID, &item.WalletID, &item.Code, &item.Name, &item.AccountType, &item.Purpose, &item.Currency,
		)
	}
	if err != nil {
		return item, err
	}
	item.Currency = strings.TrimSpace(item.Currency)
	if item.WalletID != walletID || item.AccountType != accountType || item.Purpose != purpose || item.Currency != currency {
		return finance.Account{}, platform.Invalid("account_definition_conflict", "an existing ledger account has incompatible ownership or classification")
	}
	return item, nil
}

func ensurePlatformAccount(ctx context.Context, tx pgx.Tx, tenantID, purpose, currency string) (finance.Account, error) {
	types := map[string]string{
		"cash": "asset", "revenue": "revenue", "provider_cost": "expense", "commission_expense": "expense",
	}
	accountType, ok := types[purpose]
	if !ok {
		return finance.Account{}, platform.Invalid("invalid_account_purpose", "unknown platform account purpose")
	}
	return ensureAccount(ctx, tx, tenantID, "", accountCode("platform", purpose, currency), "Platform "+strings.ReplaceAll(purpose, "_", " "), accountType, purpose, currency)
}

func accountBalance(ctx context.Context, tx pgx.Tx, tenantID, accountID string) (int64, error) {
	var balance int64
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(CASE
			WHEN account.account_type IN ('asset','expense') THEN CASE entry.direction WHEN 'debit' THEN entry.amount_minor ELSE -entry.amount_minor END
			ELSE CASE entry.direction WHEN 'credit' THEN entry.amount_minor ELSE -entry.amount_minor END
		END),0)
		FROM ledger_accounts account
		LEFT JOIN ledger_entries entry ON entry.tenant_id=account.tenant_id AND entry.account_id=account.id
		WHERE account.tenant_id=$1 AND account.id=$2
		GROUP BY account.id`, tenantID, accountID).Scan(&balance)
	return balance, err
}

func lockAccounts(ctx context.Context, tx pgx.Tx, tenantID string, accountIDs []string) error {
	ids := append([]string(nil), accountIDs...)
	sort.Strings(ids)
	last := ""
	for _, id := range ids {
		if id == last {
			continue
		}
		last = id
		var found string
		if err := tx.QueryRow(ctx, `SELECT id FROM ledger_accounts WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, id).Scan(&found); err != nil {
			return err
		}
	}
	return nil
}

func postLedger(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, key, transactionType, referenceType, referenceID, description, currency, reverses string, metadata map[string]any, inputs []finance.EntryInput) (finance.Transaction, error) {
	var item finance.Transaction
	if len(inputs) < 2 {
		return item, platform.Invalid("ledger_entries_required", "a ledger transaction requires at least two entries")
	}
	var debits, credits int64
	accountIDs := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.AmountMinor <= 0 {
			return item, platform.Invalid("invalid_amount", "ledger entry amount must be positive")
		}
		switch input.Direction {
		case "debit":
			debits += input.AmountMinor
		case "credit":
			credits += input.AmountMinor
		default:
			return item, platform.Invalid("invalid_direction", "ledger direction must be debit or credit")
		}
		accountIDs = append(accountIDs, input.AccountID)
	}
	if debits != credits {
		return item, platform.Invalid("unbalanced_transaction", fmt.Sprintf("ledger debits %d do not equal credits %d", debits, credits))
	}
	if err := lockAccounts(ctx, tx, scope.TenantID, accountIDs); err != nil {
		return item, err
	}
	for _, accountID := range accountIDs {
		var accountCurrency string
		if err := tx.QueryRow(ctx, `SELECT currency FROM ledger_accounts WHERE tenant_id=$1 AND id=$2`, scope.TenantID, accountID).Scan(&accountCurrency); err != nil {
			return item, err
		}
		if strings.TrimSpace(accountCurrency) != currency {
			return item, platform.Invalid("ledger_currency_mismatch", "all ledger accounts and entries must use the same currency")
		}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return item, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_transactions (tenant_id,idempotency_key,transaction_type,reference_type,reference_id,description,reverses_transaction_id,created_by,metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id,tenant_id,idempotency_key,transaction_type,reference_type,reference_id,description,COALESCE(reverses_transaction_id::text,''),created_by,created_at`,
		scope.TenantID, key, transactionType, referenceType, referenceID, description, nullableUUID(reverses), scope.ActorID, metadataJSON,
	).Scan(&item.ID, &item.TenantID, &item.IdempotencyKey, &item.TransactionType, &item.ReferenceType, &item.ReferenceID, &item.Description, &item.ReversesTransactionID, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	item.Metadata = metadata
	item.Entries = make([]finance.Entry, 0, len(inputs))
	for _, input := range inputs {
		var entry finance.Entry
		err = tx.QueryRow(ctx, `
			INSERT INTO ledger_entries (tenant_id,transaction_id,account_id,direction,currency,amount_minor)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id,tenant_id,transaction_id,account_id,direction,currency,amount_minor,created_at`,
			scope.TenantID, item.ID, input.AccountID, input.Direction, currency, input.AmountMinor,
		).Scan(&entry.ID, &entry.TenantID, &entry.TransactionID, &entry.AccountID, &entry.Direction, &entry.Currency, &entry.AmountMinor, &entry.CreatedAt)
		if err != nil {
			return item, err
		}
		entry.Currency = strings.TrimSpace(entry.Currency)
		item.Entries = append(item.Entries, entry)
	}
	for _, accountID := range accountIDs {
		balance, balanceErr := accountBalance(ctx, tx, scope.TenantID, accountID)
		if balanceErr != nil {
			return item, balanceErr
		}
		if _, err = tx.Exec(ctx, `INSERT INTO balance_snapshots (tenant_id,account_id,transaction_id,balance_minor) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, scope.TenantID, accountID, item.ID, balance); err != nil {
			return item, err
		}
	}
	return item, nil
}

func scanWallet(row rowScanner) (finance.Wallet, error) {
	var item finance.Wallet
	err := row.Scan(&item.ID, &item.TenantID, &item.OwnerType, &item.OwnerID, &item.Currency, &item.Status, &item.AvailableMinor, &item.HeldMinor, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func walletQuery() string {
	return `SELECT wallet.id,wallet.tenant_id,wallet.owner_type,wallet.owner_id,wallet.currency,wallet.status,
		COALESCE((SELECT sum(CASE entry.direction WHEN 'credit' THEN entry.amount_minor ELSE -entry.amount_minor END) FROM ledger_accounts account JOIN ledger_entries entry ON entry.tenant_id=account.tenant_id AND entry.account_id=account.id WHERE account.tenant_id=wallet.tenant_id AND account.wallet_id=wallet.id AND account.purpose='available'),0),
		COALESCE((SELECT sum(CASE entry.direction WHEN 'credit' THEN entry.amount_minor ELSE -entry.amount_minor END) FROM ledger_accounts account JOIN ledger_entries entry ON entry.tenant_id=account.tenant_id AND entry.account_id=account.id WHERE account.tenant_id=wallet.tenant_id AND account.wallet_id=wallet.id AND account.purpose='held'),0),
		wallet.created_by,wallet.created_at,wallet.updated_at FROM wallets wallet`
}

func getWallet(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (finance.Wallet, finance.Account, finance.Account, error) {
	query := walletQuery() + ` WHERE wallet.tenant_id=$1 AND wallet.id=$2`
	if lock {
		query += ` FOR UPDATE OF wallet`
	}
	wallet, err := scanWallet(tx.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return wallet, finance.Account{}, finance.Account{}, err
	}
	var available, held finance.Account
	err = tx.QueryRow(ctx, `SELECT id,tenant_id,COALESCE(wallet_id::text,''),code,name,account_type,purpose,currency FROM ledger_accounts WHERE tenant_id=$1 AND wallet_id=$2 AND purpose='available'`, tenantID, id).Scan(
		&available.ID, &available.TenantID, &available.WalletID, &available.Code, &available.Name, &available.AccountType, &available.Purpose, &available.Currency,
	)
	if err != nil {
		return wallet, available, held, err
	}
	err = tx.QueryRow(ctx, `SELECT id,tenant_id,COALESCE(wallet_id::text,''),code,name,account_type,purpose,currency FROM ledger_accounts WHERE tenant_id=$1 AND wallet_id=$2 AND purpose='held'`, tenantID, id).Scan(
		&held.ID, &held.TenantID, &held.WalletID, &held.Code, &held.Name, &held.AccountType, &held.Purpose, &held.Currency,
	)
	available.Currency = strings.TrimSpace(available.Currency)
	held.Currency = strings.TrimSpace(held.Currency)
	return wallet, available, held, err
}

func (s *Store) CreateWallet(ctx context.Context, scope tenancy.Scope, input application.WalletInput, key string) (finance.Wallet, error) {
	input.OwnerType = strings.ToLower(strings.TrimSpace(input.OwnerType))
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	currency, err := normalizeCurrency(input.Currency)
	if err != nil {
		return finance.Wallet{}, err
	}
	if input.OwnerID == "" || (input.OwnerType != "customer" && input.OwnerType != "provider" && input.OwnerType != "reseller" && input.OwnerType != "platform") {
		return finance.Wallet{}, platform.Invalid("invalid_wallet_owner", "wallet owner type and identifier are required")
	}
	return runCommand(ctx, s, scope, key, "wallet.create", func(tx pgx.Tx) (finance.Wallet, string, error) {
		var item finance.Wallet
		err = tx.QueryRow(ctx, `INSERT INTO wallets (tenant_id,owner_type,owner_id,currency,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,owner_type,owner_id,currency,status,0,0,created_by,created_at,updated_at`, scope.TenantID, input.OwnerType, input.OwnerID, currency, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.OwnerType, &item.OwnerID, &item.Currency, &item.Status, &item.AvailableMinor, &item.HeldMinor, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		if _, err = ensureAccount(ctx, tx, scope.TenantID, item.ID, accountCode("wallet", item.ID, "available", currency), "Wallet available", "liability", "available", currency); err != nil {
			return item, "", err
		}
		if _, err = ensureAccount(ctx, tx, scope.TenantID, item.ID, accountCode("wallet", item.ID, "held", currency), "Wallet held", "liability", "held", currency); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"owner_type": item.OwnerType, "owner_id": item.OwnerID, "currency": item.Currency}
		if err = appendAudit(ctx, tx, scope, "wallet.create", "wallet", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "wallet", item.ID, "wallet.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) PostWalletAdjustment(ctx context.Context, scope tenancy.Scope, walletID string, input application.WalletAdjustmentInput, key string) (finance.Adjustment, error) {
	input.Direction = strings.ToLower(strings.TrimSpace(input.Direction))
	input.Reason = strings.TrimSpace(input.Reason)
	if input.AmountMinor <= 0 || input.Reason == "" || (input.Direction != "credit" && input.Direction != "debit") {
		return finance.Adjustment{}, platform.Invalid("invalid_adjustment", "positive amount, credit or debit direction, and reason are required")
	}
	return runCommand(ctx, s, scope, key, "wallet.adjust", func(tx pgx.Tx) (finance.Adjustment, string, error) {
		wallet, available, _, err := getWallet(ctx, tx, scope.TenantID, walletID, true)
		if err != nil {
			return finance.Adjustment{}, "", err
		}
		if wallet.Status != "active" {
			return finance.Adjustment{}, "", platform.Invalid("wallet_unavailable", "wallet must be active")
		}
		cash, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "cash", wallet.Currency)
		if err != nil {
			return finance.Adjustment{}, "", err
		}
		if input.Direction == "debit" && wallet.AvailableMinor < input.AmountMinor {
			return finance.Adjustment{}, "", platform.Invalid("insufficient_wallet_balance", "wallet available balance is insufficient")
		}
		entries := []finance.EntryInput{{AccountID: cash.ID, Direction: "debit", AmountMinor: input.AmountMinor}, {AccountID: available.ID, Direction: "credit", AmountMinor: input.AmountMinor}}
		if input.Direction == "debit" {
			entries[0].Direction, entries[1].Direction = "credit", "debit"
		}
		transaction, err := postLedger(ctx, tx, scope, key, "adjustment", "wallet", wallet.ID, input.Reason, wallet.Currency, "", map[string]any{"direction": input.Direction}, entries)
		if err != nil {
			return finance.Adjustment{}, "", err
		}
		var item finance.Adjustment
		err = tx.QueryRow(ctx, `INSERT INTO financial_adjustments (tenant_id,wallet_id,direction,currency,amount_minor,reason,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,wallet_id,direction,currency,amount_minor,reason,ledger_transaction_id,idempotency_key,created_by,created_at`, scope.TenantID, wallet.ID, input.Direction, wallet.Currency, input.AmountMinor, input.Reason, transaction.ID, key, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.WalletID, &item.Direction, &item.Currency, &item.AmountMinor, &item.Reason, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		metadata := map[string]any{"wallet_id": wallet.ID, "direction": item.Direction, "amount_minor": item.AmountMinor, "currency": item.Currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "wallet.adjust", "financial_adjustment", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "wallet", wallet.ID, "wallet.adjusted", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanHold(row rowScanner) (finance.Hold, error) {
	var item finance.Hold
	err := row.Scan(&item.ID, &item.TenantID, &item.WalletID, &item.OrderID, &item.Currency, &item.AmountMinor, &item.CapturedMinor, &item.ReleasedMinor, &item.RemainingMinor, &item.Status, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

const holdSelect = `SELECT id,tenant_id,wallet_id,order_id,currency,amount_minor,captured_minor,released_minor,amount_minor-captured_minor-released_minor,status,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at FROM holds`

func (s *Store) PlaceOrderHold(ctx context.Context, scope tenancy.Scope, orderID string, input application.HoldInput, key string) (finance.Hold, error) {
	if input.WalletID == "" || input.AmountMinor <= 0 {
		return finance.Hold{}, platform.Invalid("invalid_hold", "wallet and positive hold amount are required")
	}
	return runCommand(ctx, s, scope, key, "hold.create", func(tx pgx.Tx) (finance.Hold, string, error) {
		var orderCustomer, orderCurrency, orderStatus string
		if err := tx.QueryRow(ctx, `SELECT customer_id,currency,status FROM orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, orderID).Scan(&orderCustomer, &orderCurrency, &orderStatus); err != nil {
			return finance.Hold{}, "", err
		}
		orderCurrency = strings.TrimSpace(orderCurrency)
		if orderStatus != "created" && orderStatus != "awaiting_payment" {
			return finance.Hold{}, "", platform.Invalid("order_not_payable", "funds can only be held for a created or awaiting-payment order")
		}
		wallet, available, held, err := getWallet(ctx, tx, scope.TenantID, input.WalletID, true)
		if err != nil {
			return finance.Hold{}, "", err
		}
		if wallet.Status != "active" || wallet.Currency != orderCurrency || wallet.OwnerType != "customer" || wallet.OwnerID != orderCustomer {
			return finance.Hold{}, "", platform.Invalid("wallet_order_mismatch", "an active customer wallet with the order currency and customer is required")
		}
		if wallet.AvailableMinor < input.AmountMinor {
			return finance.Hold{}, "", platform.Invalid("insufficient_wallet_balance", "wallet available balance is insufficient for the hold")
		}
		transaction, err := postLedger(ctx, tx, scope, key, "hold", "order", orderID, "Reserve order funds", wallet.Currency, "", map[string]any{"wallet_id": wallet.ID}, []finance.EntryInput{
			{AccountID: available.ID, Direction: "debit", AmountMinor: input.AmountMinor},
			{AccountID: held.ID, Direction: "credit", AmountMinor: input.AmountMinor},
		})
		if err != nil {
			return finance.Hold{}, "", err
		}
		item, err := scanHold(tx.QueryRow(ctx, `INSERT INTO holds (tenant_id,wallet_id,order_id,currency,amount_minor,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,wallet_id,order_id,currency,amount_minor,captured_minor,released_minor,amount_minor-captured_minor-released_minor,status,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at`, scope.TenantID, wallet.ID, orderID, wallet.Currency, input.AmountMinor, transaction.ID, key, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"order_id": orderID, "wallet_id": wallet.ID, "amount_minor": item.AmountMinor, "currency": item.Currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "hold.create", "hold", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "hold", item.ID, "funds.held", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReleaseHold(ctx context.Context, scope tenancy.Scope, holdID string, input application.ReleaseInput, key string) (finance.Hold, error) {
	if input.AmountMinor < 0 {
		return finance.Hold{}, platform.Invalid("invalid_release", "release amount cannot be negative")
	}
	return runCommand(ctx, s, scope, key, "hold.release", func(tx pgx.Tx) (finance.Hold, string, error) {
		item, err := scanHold(tx.QueryRow(ctx, holdSelect+` WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, holdID))
		if err != nil {
			return item, "", err
		}
		amount := input.AmountMinor
		if amount == 0 {
			amount = item.RemainingMinor
		}
		if amount <= 0 || amount > item.RemainingMinor {
			return item, "", platform.Invalid("invalid_release", "release amount exceeds the remaining hold")
		}
		_, available, held, err := getWallet(ctx, tx, scope.TenantID, item.WalletID, true)
		if err != nil {
			return item, "", err
		}
		transaction, err := postLedger(ctx, tx, scope, key, "release", "hold", item.ID, "Release reserved funds", item.Currency, "", map[string]any{"order_id": item.OrderID}, []finance.EntryInput{
			{AccountID: held.ID, Direction: "debit", AmountMinor: amount},
			{AccountID: available.ID, Direction: "credit", AmountMinor: amount},
		})
		if err != nil {
			return item, "", err
		}
		newRemaining := item.RemainingMinor - amount
		status := "active"
		if newRemaining == 0 {
			if item.CapturedMinor > 0 {
				status = "captured"
			} else {
				status = "released"
			}
		} else if item.CapturedMinor > 0 {
			status = "partially_captured"
		}
		if _, err = tx.Exec(ctx, `UPDATE holds SET released_minor=released_minor+$3,status=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, item.ID, amount, status); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO hold_releases (tenant_id,hold_id,amount_minor,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6)`, scope.TenantID, item.ID, amount, transaction.ID, key, scope.ActorID); err != nil {
			return item, "", err
		}
		item, err = scanHold(tx.QueryRow(ctx, holdSelect+` WHERE tenant_id=$1 AND id=$2`, scope.TenantID, holdID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"amount_minor": amount, "remaining_minor": item.RemainingMinor, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "hold.release", "hold", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "hold", item.ID, "funds.released", 2, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) PostCustomerCharge(ctx context.Context, scope tenancy.Scope, chargeID string, input application.ChargePostingInput, key string) (finance.Transaction, error) {
	if input.HoldID == "" {
		return finance.Transaction{}, platform.Invalid("hold_required", "a hold is required to post a customer charge")
	}
	return runCommand(ctx, s, scope, key, "customer_charge.post", func(tx pgx.Tx) (finance.Transaction, string, error) {
		var orderID, currency, status string
		var amountMinor int64
		if err := tx.QueryRow(ctx, `SELECT order_id,currency,amount_minor,status FROM customer_charges WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, chargeID).Scan(&orderID, &currency, &amountMinor, &status); err != nil {
			return finance.Transaction{}, "", err
		}
		currency = strings.TrimSpace(currency)
		if status != "calculated" {
			return finance.Transaction{}, "", platform.Invalid("charge_not_postable", "customer charge must be calculated and unposted")
		}
		if amountMinor <= 0 {
			return finance.Transaction{}, "", platform.Invalid("zero_charge", "a zero-value customer charge has no ledger effect")
		}
		hold, err := scanHold(tx.QueryRow(ctx, holdSelect+` WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.HoldID))
		if err != nil {
			return finance.Transaction{}, "", err
		}
		if hold.OrderID != orderID || hold.Currency != currency || hold.RemainingMinor < amountMinor || (hold.Status != "active" && hold.Status != "partially_captured") {
			return finance.Transaction{}, "", platform.Invalid("hold_charge_mismatch", "active held funds for the same order and currency must cover the charge")
		}
		_, _, held, err := getWallet(ctx, tx, scope.TenantID, hold.WalletID, true)
		if err != nil {
			return finance.Transaction{}, "", err
		}
		revenue, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "revenue", currency)
		if err != nil {
			return finance.Transaction{}, "", err
		}
		transaction, err := postLedger(ctx, tx, scope, key, "charge", "customer_charge", chargeID, "Post customer charge", currency, "", map[string]any{"order_id": orderID, "hold_id": hold.ID}, []finance.EntryInput{
			{AccountID: held.ID, Direction: "debit", AmountMinor: amountMinor},
			{AccountID: revenue.ID, Direction: "credit", AmountMinor: amountMinor},
		})
		if err != nil {
			return finance.Transaction{}, "", err
		}
		newRemaining := hold.RemainingMinor - amountMinor
		holdStatus := "partially_captured"
		if newRemaining == 0 {
			holdStatus = "captured"
		}
		if _, err = tx.Exec(ctx, `UPDATE holds SET captured_minor=captured_minor+$3,status=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, hold.ID, amountMinor, holdStatus); err != nil {
			return finance.Transaction{}, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE customer_charges SET status='posted',ledger_transaction_id=$3,posted_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, chargeID, transaction.ID); err != nil {
			return finance.Transaction{}, "", err
		}
		metadata := map[string]any{"order_id": orderID, "hold_id": hold.ID, "amount_minor": amountMinor, "currency": currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "customer_charge.post", "customer_charge", chargeID, key, metadata); err != nil {
			return finance.Transaction{}, "", err
		}
		if err = appendEvent(ctx, tx, scope, "customer_charge", chargeID, "customer_charge.posted", 2, metadata); err != nil {
			return finance.Transaction{}, "", err
		}
		return transaction, transaction.ID, nil
	})
}

func (s *Store) RefundCustomerCharge(ctx context.Context, scope tenancy.Scope, chargeID string, input application.RefundInput, key string) (finance.Refund, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.WalletID == "" || input.AmountMinor <= 0 || input.Reason == "" {
		return finance.Refund{}, platform.Invalid("invalid_refund", "wallet, positive amount, and reason are required")
	}
	return runCommand(ctx, s, scope, key, "customer_charge.refund", func(tx pgx.Tx) (finance.Refund, string, error) {
		var orderID, customerID, currency, status, originalTransactionID string
		var chargeMinor, refundedMinor int64
		err := tx.QueryRow(ctx, `
			SELECT charge.order_id,orders.customer_id,charge.currency,charge.amount_minor,charge.status,COALESCE(charge.ledger_transaction_id::text,''),
			       COALESCE((SELECT sum(refund.amount_minor) FROM refunds refund WHERE refund.tenant_id=charge.tenant_id AND refund.customer_charge_id=charge.id AND refund.status='posted'),0)
			FROM customer_charges charge JOIN orders ON orders.tenant_id=charge.tenant_id AND orders.id=charge.order_id
			WHERE charge.tenant_id=$1 AND charge.id=$2 FOR UPDATE OF charge`, scope.TenantID, chargeID).Scan(&orderID, &customerID, &currency, &chargeMinor, &status, &originalTransactionID, &refundedMinor)
		if err != nil {
			return finance.Refund{}, "", err
		}
		currency = strings.TrimSpace(currency)
		if status != "posted" || originalTransactionID == "" || input.AmountMinor > chargeMinor-refundedMinor {
			return finance.Refund{}, "", platform.Invalid("refund_not_allowed", "posted charge and unrefunded amount must cover the refund")
		}
		wallet, available, _, err := getWallet(ctx, tx, scope.TenantID, input.WalletID, true)
		if err != nil {
			return finance.Refund{}, "", err
		}
		if wallet.Status != "active" || wallet.OwnerType != "customer" || wallet.OwnerID != customerID || wallet.Currency != currency {
			return finance.Refund{}, "", platform.Invalid("refund_wallet_mismatch", "refund wallet must be the active wallet for the charged customer and currency")
		}
		revenue, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "revenue", currency)
		if err != nil {
			return finance.Refund{}, "", err
		}
		transaction, err := postLedger(ctx, tx, scope, key, "refund", "customer_charge", chargeID, input.Reason, currency, "", map[string]any{"order_id": orderID, "original_transaction_id": originalTransactionID}, []finance.EntryInput{
			{AccountID: revenue.ID, Direction: "debit", AmountMinor: input.AmountMinor},
			{AccountID: available.ID, Direction: "credit", AmountMinor: input.AmountMinor},
		})
		if err != nil {
			return finance.Refund{}, "", err
		}
		var item finance.Refund
		err = tx.QueryRow(ctx, `INSERT INTO refunds (tenant_id,customer_charge_id,wallet_id,currency,amount_minor,reason,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,customer_charge_id,wallet_id,currency,amount_minor,reason,status,ledger_transaction_id,idempotency_key,created_by,created_at`, scope.TenantID, chargeID, wallet.ID, currency, input.AmountMinor, input.Reason, transaction.ID, key, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.CustomerChargeID, &item.WalletID, &item.Currency, &item.AmountMinor, &item.Reason, &item.Status, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		if refundedMinor+input.AmountMinor == chargeMinor {
			if _, err = tx.Exec(ctx, `UPDATE customer_charges SET status='reversed' WHERE tenant_id=$1 AND id=$2`, scope.TenantID, chargeID); err != nil {
				return item, "", err
			}
		}
		metadata := map[string]any{"customer_charge_id": chargeID, "wallet_id": wallet.ID, "amount_minor": item.AmountMinor, "currency": item.Currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "customer_charge.refund", "refund", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "customer_charge", chargeID, "customer_charge.refunded", 3, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateCommission(ctx context.Context, scope tenancy.Scope, chargeID string, input application.CommissionInput, key string) (finance.Commission, error) {
	input.BeneficiaryType = strings.ToLower(strings.TrimSpace(input.BeneficiaryType))
	input.BeneficiaryID = strings.TrimSpace(input.BeneficiaryID)
	if input.AmountMinor <= 0 || input.BeneficiaryID == "" || (input.BeneficiaryType != "reseller" && input.BeneficiaryType != "agent" && input.BeneficiaryType != "partner") {
		return finance.Commission{}, platform.Invalid("invalid_commission", "beneficiary and positive integer commission amount are required")
	}
	return runCommand(ctx, s, scope, key, "commission.create", func(tx pgx.Tx) (finance.Commission, string, error) {
		var currency, chargeStatus string
		var chargeMinor, committedMinor int64
		err := tx.QueryRow(ctx, `SELECT currency,amount_minor,status,COALESCE((SELECT sum(amount_minor) FROM commissions WHERE tenant_id=$1 AND customer_charge_id=$2 AND status<>'reversed'),0) FROM customer_charges WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, chargeID).Scan(&currency, &chargeMinor, &chargeStatus, &committedMinor)
		if err != nil {
			return finance.Commission{}, "", err
		}
		currency = strings.TrimSpace(currency)
		if chargeStatus != "posted" || committedMinor+input.AmountMinor > chargeMinor {
			return finance.Commission{}, "", platform.Invalid("commission_not_allowed", "posted charge amount must cover total commissions")
		}
		expense, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "commission_expense", currency)
		if err != nil {
			return finance.Commission{}, "", err
		}
		payable, err := ensureAccount(ctx, tx, scope.TenantID, "", accountCode(input.BeneficiaryType, input.BeneficiaryID, "commission_payable", currency), "Commission payable", "liability", "commission_payable", currency)
		if err != nil {
			return finance.Commission{}, "", err
		}
		transaction, err := postLedger(ctx, tx, scope, key, "commission", "customer_charge", chargeID, "Record commission payable", currency, "", map[string]any{"beneficiary_type": input.BeneficiaryType, "beneficiary_id": input.BeneficiaryID}, []finance.EntryInput{
			{AccountID: expense.ID, Direction: "debit", AmountMinor: input.AmountMinor},
			{AccountID: payable.ID, Direction: "credit", AmountMinor: input.AmountMinor},
		})
		if err != nil {
			return finance.Commission{}, "", err
		}
		var item finance.Commission
		err = tx.QueryRow(ctx, `INSERT INTO commissions (tenant_id,customer_charge_id,beneficiary_type,beneficiary_id,currency,amount_minor,payable_account_id,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,tenant_id,customer_charge_id,beneficiary_type,beneficiary_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at`, scope.TenantID, chargeID, input.BeneficiaryType, input.BeneficiaryID, currency, input.AmountMinor, payable.ID, transaction.ID, key, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.CustomerChargeID, &item.BeneficiaryType, &item.BeneficiaryID, &item.Currency, &item.AmountMinor, &item.SettledMinor, &item.Status, &item.PayableAccountID, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		metadata := map[string]any{"customer_charge_id": chargeID, "beneficiary_type": item.BeneficiaryType, "beneficiary_id": item.BeneficiaryID, "amount_minor": item.AmountMinor, "currency": item.Currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "commission.create", "commission", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "commission", item.ID, "commission.recorded", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateProviderPayable(ctx context.Context, scope tenancy.Scope, providerCostID, key string) (finance.ProviderPayable, error) {
	return runCommand(ctx, s, scope, key, "provider_payable.create", func(tx pgx.Tx) (finance.ProviderPayable, string, error) {
		var providerID, currency string
		var amountMinor int64
		err := tx.QueryRow(ctx, `SELECT endpoint.provider_id,cost.currency,cost.amount_minor FROM provider_costs cost JOIN provider_endpoints endpoint ON endpoint.tenant_id=cost.tenant_id AND endpoint.id=cost.provider_endpoint_id WHERE cost.tenant_id=$1 AND cost.id=$2 FOR UPDATE OF cost`, scope.TenantID, providerCostID).Scan(&providerID, &currency, &amountMinor)
		if err != nil {
			return finance.ProviderPayable{}, "", err
		}
		currency = strings.TrimSpace(currency)
		if amountMinor <= 0 {
			return finance.ProviderPayable{}, "", platform.Invalid("zero_provider_cost", "a zero-value provider cost has no payable ledger effect")
		}
		expense, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "provider_cost", currency)
		if err != nil {
			return finance.ProviderPayable{}, "", err
		}
		payable, err := ensureAccount(ctx, tx, scope.TenantID, "", accountCode("provider", providerID, "payable", currency), "Provider payable", "liability", "provider_payable", currency)
		if err != nil {
			return finance.ProviderPayable{}, "", err
		}
		transaction, err := postLedger(ctx, tx, scope, key, "provider_payable", "provider_cost", providerCostID, "Record provider payable", currency, "", map[string]any{"provider_id": providerID}, []finance.EntryInput{
			{AccountID: expense.ID, Direction: "debit", AmountMinor: amountMinor},
			{AccountID: payable.ID, Direction: "credit", AmountMinor: amountMinor},
		})
		if err != nil {
			return finance.ProviderPayable{}, "", err
		}
		var item finance.ProviderPayable
		err = tx.QueryRow(ctx, `INSERT INTO provider_payables (tenant_id,provider_cost_id,provider_id,currency,amount_minor,payable_account_id,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,provider_cost_id,provider_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at`, scope.TenantID, providerCostID, providerID, currency, amountMinor, payable.ID, transaction.ID, key, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.ProviderCostID, &item.ProviderID, &item.Currency, &item.AmountMinor, &item.SettledMinor, &item.Status, &item.PayableAccountID, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		metadata := map[string]any{"provider_cost_id": providerCostID, "provider_id": providerID, "amount_minor": amountMinor, "currency": currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "provider_payable.create", "provider_payable", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "provider_payable", item.ID, "provider_payable.recorded", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateSettlement(ctx context.Context, scope tenancy.Scope, input application.SettlementInput, key string) (finance.Settlement, error) {
	input.SourceType = strings.ToLower(strings.TrimSpace(input.SourceType))
	if input.SourceID == "" || input.AmountMinor < 0 || (input.SourceType != "provider_payable" && input.SourceType != "commission") {
		return finance.Settlement{}, platform.Invalid("invalid_settlement", "provider payable or commission source and non-negative amount are required")
	}
	return runCommand(ctx, s, scope, key, "settlement.create", func(tx pgx.Tx) (finance.Settlement, string, error) {
		var beneficiaryType, beneficiaryID, currency, status, payableAccountID string
		var amountMinor, settledMinor int64
		var err error
		if input.SourceType == "provider_payable" {
			err = tx.QueryRow(ctx, `SELECT 'provider',provider_id::text,currency,amount_minor,settled_minor,status,payable_account_id FROM provider_payables WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.SourceID).Scan(&beneficiaryType, &beneficiaryID, &currency, &amountMinor, &settledMinor, &status, &payableAccountID)
		} else {
			err = tx.QueryRow(ctx, `SELECT beneficiary_type,beneficiary_id,currency,amount_minor,settled_minor,status,payable_account_id FROM commissions WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.SourceID).Scan(&beneficiaryType, &beneficiaryID, &currency, &amountMinor, &settledMinor, &status, &payableAccountID)
		}
		if err != nil {
			return finance.Settlement{}, "", err
		}
		currency = strings.TrimSpace(currency)
		outstanding := amountMinor - settledMinor
		settlementMinor := input.AmountMinor
		if settlementMinor == 0 {
			settlementMinor = outstanding
		}
		if status == "settled" || status == "reversed" || settlementMinor <= 0 || settlementMinor > outstanding {
			return finance.Settlement{}, "", platform.Invalid("settlement_not_allowed", "open payable amount must cover the settlement")
		}
		cash, err := ensurePlatformAccount(ctx, tx, scope.TenantID, "cash", currency)
		if err != nil {
			return finance.Settlement{}, "", err
		}
		if err = lockAccounts(ctx, tx, scope.TenantID, []string{cash.ID, payableAccountID}); err != nil {
			return finance.Settlement{}, "", err
		}
		cashBalance, err := accountBalance(ctx, tx, scope.TenantID, cash.ID)
		if err != nil {
			return finance.Settlement{}, "", err
		}
		if cashBalance < settlementMinor {
			return finance.Settlement{}, "", platform.Invalid("insufficient_platform_cash", "platform cash balance is insufficient for settlement")
		}
		transaction, err := postLedger(ctx, tx, scope, key, "settlement", input.SourceType, input.SourceID, "Settle payable", currency, "", map[string]any{"beneficiary_type": beneficiaryType, "beneficiary_id": beneficiaryID}, []finance.EntryInput{
			{AccountID: payableAccountID, Direction: "debit", AmountMinor: settlementMinor},
			{AccountID: cash.ID, Direction: "credit", AmountMinor: settlementMinor},
		})
		if err != nil {
			return finance.Settlement{}, "", err
		}
		var item finance.Settlement
		err = tx.QueryRow(ctx, `INSERT INTO settlements (tenant_id,source_type,source_id,beneficiary_type,beneficiary_id,currency,amount_minor,ledger_transaction_id,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,tenant_id,source_type,source_id,beneficiary_type,beneficiary_id,currency,amount_minor,status,ledger_transaction_id,idempotency_key,created_by,created_at`, scope.TenantID, input.SourceType, input.SourceID, beneficiaryType, beneficiaryID, currency, settlementMinor, transaction.ID, key, scope.ActorID).Scan(
			&item.ID, &item.TenantID, &item.SourceType, &item.SourceID, &item.BeneficiaryType, &item.BeneficiaryID, &item.Currency, &item.AmountMinor, &item.Status, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		newSettled := settledMinor + settlementMinor
		newStatus := "partially_settled"
		if newSettled == amountMinor {
			newStatus = "settled"
		}
		if input.SourceType == "provider_payable" {
			_, err = tx.Exec(ctx, `UPDATE provider_payables SET settled_minor=$3,status=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.SourceID, newSettled, newStatus)
		} else {
			_, err = tx.Exec(ctx, `UPDATE commissions SET settled_minor=$3,status=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.SourceID, newSettled, newStatus)
		}
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"source_type": input.SourceType, "source_id": input.SourceID, "amount_minor": settlementMinor, "currency": currency, "ledger_transaction_id": transaction.ID}
		if err = appendAudit(ctx, tx, scope, "settlement.create", "settlement", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "settlement", item.ID, "settlement.completed", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

type reconciliationFact struct {
	ReferenceType string
	ReferenceID   string
	Currency      string
	ExpectedMinor int64
	ActualMinor   int64
}

func loadReconciliationFacts(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]reconciliationFact, error) {
	facts := []reconciliationFact{}
	orderFilter := ""
	args := []any{tenantID}
	if orderID != "" {
		orderFilter = " AND charge.order_id=$2"
		args = append(args, orderID)
	}
	rows, err := tx.Query(ctx, `
		SELECT 'customer_charge',charge.id,charge.currency,charge.amount_minor,
		       COALESCE((SELECT sum(entry.amount_minor) FROM ledger_entries entry WHERE entry.tenant_id=charge.tenant_id AND entry.transaction_id=charge.ledger_transaction_id AND entry.direction='credit'),0)
		FROM customer_charges charge WHERE charge.tenant_id=$1`+orderFilter+` ORDER BY charge.created_at,charge.id`, args...)
	if err != nil {
		return facts, err
	}
	for rows.Next() {
		var fact reconciliationFact
		if err = rows.Scan(&fact.ReferenceType, &fact.ReferenceID, &fact.Currency, &fact.ExpectedMinor, &fact.ActualMinor); err != nil {
			rows.Close()
			return facts, err
		}
		fact.Currency = strings.TrimSpace(fact.Currency)
		facts = append(facts, fact)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return facts, err
	}
	rows.Close()

	orderFilter = ""
	args = []any{tenantID}
	if orderID != "" {
		orderFilter = " AND cost.order_id=$2"
		args = append(args, orderID)
	}
	rows, err = tx.Query(ctx, `
		SELECT 'provider_cost',cost.id,cost.currency,cost.amount_minor,COALESCE(payable.amount_minor,0)
		FROM provider_costs cost LEFT JOIN provider_payables payable ON payable.tenant_id=cost.tenant_id AND payable.provider_cost_id=cost.id
		WHERE cost.tenant_id=$1`+orderFilter+` ORDER BY cost.created_at,cost.id`, args...)
	if err != nil {
		return facts, err
	}
	for rows.Next() {
		var fact reconciliationFact
		if err = rows.Scan(&fact.ReferenceType, &fact.ReferenceID, &fact.Currency, &fact.ExpectedMinor, &fact.ActualMinor); err != nil {
			rows.Close()
			return facts, err
		}
		fact.Currency = strings.TrimSpace(fact.Currency)
		facts = append(facts, fact)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return facts, err
	}
	rows.Close()

	orderFilter = ""
	args = []any{tenantID}
	if orderID != "" {
		orderFilter = " AND charge.order_id=$2"
		args = append(args, orderID)
	}
	rows, err = tx.Query(ctx, `
		SELECT 'commission',commission.id,commission.currency,commission.amount_minor,
		       COALESCE((SELECT sum(entry.amount_minor) FROM ledger_entries entry WHERE entry.tenant_id=commission.tenant_id AND entry.transaction_id=commission.ledger_transaction_id AND entry.direction='credit'),0)
		FROM commissions commission JOIN customer_charges charge ON charge.tenant_id=commission.tenant_id AND charge.id=commission.customer_charge_id
		WHERE commission.tenant_id=$1 AND commission.status<>'reversed'`+orderFilter+` ORDER BY commission.created_at,commission.id`, args...)
	if err != nil {
		return facts, err
	}
	defer rows.Close()
	for rows.Next() {
		var fact reconciliationFact
		if err = rows.Scan(&fact.ReferenceType, &fact.ReferenceID, &fact.Currency, &fact.ExpectedMinor, &fact.ActualMinor); err != nil {
			return facts, err
		}
		fact.Currency = strings.TrimSpace(fact.Currency)
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func (s *Store) RunReconciliation(ctx context.Context, scope tenancy.Scope, input application.ReconciliationInput, key string) (finance.ReconciliationRun, error) {
	return runCommand(ctx, s, scope, key, "reconciliation.run", func(tx pgx.Tx) (finance.ReconciliationRun, string, error) {
		if input.OrderID != "" {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE tenant_id=$1 AND id=$2)`, scope.TenantID, input.OrderID).Scan(&exists); err != nil {
				return finance.ReconciliationRun{}, "", err
			}
			if !exists {
				return finance.ReconciliationRun{}, "", platform.Invalid("not_found", "order not found")
			}
		}
		facts, err := loadReconciliationFacts(ctx, tx, scope.TenantID, input.OrderID)
		if err != nil {
			return finance.ReconciliationRun{}, "", err
		}
		var item finance.ReconciliationRun
		err = tx.QueryRow(ctx, `INSERT INTO reconciliation_runs (tenant_id,order_id,status,checked_count,discrepancy_count,created_by,idempotency_key) VALUES($1,$2,'matched',0,0,$3,$4) RETURNING id,tenant_id,COALESCE(order_id::text,''),status,checked_count,discrepancy_count,created_by,idempotency_key,started_at,completed_at`, scope.TenantID, nullableUUID(input.OrderID), scope.ActorID, key).Scan(
			&item.ID, &item.TenantID, &item.OrderID, &item.Status, &item.CheckedCount, &item.DiscrepancyCount, &item.CreatedBy, &item.IdempotencyKey, &item.StartedAt, &item.CompletedAt,
		)
		if err != nil {
			return item, "", err
		}
		item.Items = []finance.ReconciliationItem{}
		for _, fact := range facts {
			status := "matched"
			if fact.ExpectedMinor != fact.ActualMinor {
				status = "discrepancy"
				item.DiscrepancyCount++
			}
			var reconciliationItem finance.ReconciliationItem
			err = tx.QueryRow(ctx, `INSERT INTO reconciliation_items (tenant_id,run_id,reference_type,reference_id,currency,expected_minor,actual_minor,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,run_id,reference_type,reference_id,currency,expected_minor,actual_minor,status,created_at`, scope.TenantID, item.ID, fact.ReferenceType, fact.ReferenceID, fact.Currency, fact.ExpectedMinor, fact.ActualMinor, status).Scan(
				&reconciliationItem.ID, &reconciliationItem.TenantID, &reconciliationItem.RunID, &reconciliationItem.ReferenceType, &reconciliationItem.ReferenceID, &reconciliationItem.Currency, &reconciliationItem.ExpectedMinor, &reconciliationItem.ActualMinor, &reconciliationItem.Status, &reconciliationItem.CreatedAt,
			)
			if err != nil {
				return item, "", err
			}
			reconciliationItem.Currency = strings.TrimSpace(reconciliationItem.Currency)
			item.Items = append(item.Items, reconciliationItem)
			if status == "discrepancy" {
				difference := fact.ActualMinor - fact.ExpectedMinor
				if _, err = tx.Exec(ctx, `INSERT INTO reconciliation_discrepancies (tenant_id,run_id,item_id,discrepancy_type,difference_minor) VALUES($1,$2,$3,'amount_mismatch',$4)`, scope.TenantID, item.ID, reconciliationItem.ID, difference); err != nil {
					return item, "", err
				}
			}
		}
		item.CheckedCount = len(item.Items)
		if item.DiscrepancyCount > 0 {
			item.Status = "discrepancy"
		}
		if err = tx.QueryRow(ctx, `UPDATE reconciliation_runs SET status=$3,checked_count=$4,discrepancy_count=$5,completed_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING completed_at`, scope.TenantID, item.ID, item.Status, item.CheckedCount, item.DiscrepancyCount).Scan(&item.CompletedAt); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"order_id": item.OrderID, "status": item.Status, "checked_count": item.CheckedCount, "discrepancy_count": item.DiscrepancyCount}
		if err = appendAudit(ctx, tx, scope, "reconciliation.run", "reconciliation_run", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "reconciliation_run", item.ID, "reconciliation.completed", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanRefund(row rowScanner) (finance.Refund, error) {
	var item finance.Refund
	err := row.Scan(&item.ID, &item.TenantID, &item.CustomerChargeID, &item.WalletID, &item.Currency, &item.AmountMinor, &item.Reason, &item.Status, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanCommission(row rowScanner) (finance.Commission, error) {
	var item finance.Commission
	err := row.Scan(&item.ID, &item.TenantID, &item.CustomerChargeID, &item.BeneficiaryType, &item.BeneficiaryID, &item.Currency, &item.AmountMinor, &item.SettledMinor, &item.Status, &item.PayableAccountID, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanProviderPayable(row rowScanner) (finance.ProviderPayable, error) {
	var item finance.ProviderPayable
	err := row.Scan(&item.ID, &item.TenantID, &item.ProviderCostID, &item.ProviderID, &item.Currency, &item.AmountMinor, &item.SettledMinor, &item.Status, &item.PayableAccountID, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanSettlement(row rowScanner) (finance.Settlement, error) {
	var item finance.Settlement
	err := row.Scan(&item.ID, &item.TenantID, &item.SourceType, &item.SourceID, &item.BeneficiaryType, &item.BeneficiaryID, &item.Currency, &item.AmountMinor, &item.Status, &item.LedgerTransactionID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func listRows[T any](rows pgx.Rows, scan func(row rowScanner) (T, error)) ([]T, error) {
	defer rows.Close()
	items := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func listReconciliationRuns(ctx context.Context, tx pgx.Tx, tenantID string) ([]finance.ReconciliationRun, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,COALESCE(order_id::text,''),status,checked_count,discrepancy_count,created_by,idempotency_key,started_at,completed_at FROM reconciliation_runs WHERE tenant_id=$1 ORDER BY completed_at DESC,id DESC LIMIT 50`, tenantID)
	if err != nil {
		return nil, err
	}
	runs := []finance.ReconciliationRun{}
	for rows.Next() {
		var run finance.ReconciliationRun
		if err = rows.Scan(&run.ID, &run.TenantID, &run.OrderID, &run.Status, &run.CheckedCount, &run.DiscrepancyCount, &run.CreatedBy, &run.IdempotencyKey, &run.StartedAt, &run.CompletedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range runs {
		runs[index].Items = []finance.ReconciliationItem{}
		itemRows, itemErr := tx.Query(ctx, `SELECT id,tenant_id,run_id,reference_type,reference_id,currency,expected_minor,actual_minor,status,created_at FROM reconciliation_items WHERE tenant_id=$1 AND run_id=$2 ORDER BY created_at,id`, tenantID, runs[index].ID)
		if itemErr != nil {
			return nil, itemErr
		}
		for itemRows.Next() {
			var item finance.ReconciliationItem
			if itemErr = itemRows.Scan(&item.ID, &item.TenantID, &item.RunID, &item.ReferenceType, &item.ReferenceID, &item.Currency, &item.ExpectedMinor, &item.ActualMinor, &item.Status, &item.CreatedAt); itemErr != nil {
				itemRows.Close()
				return nil, itemErr
			}
			item.Currency = strings.TrimSpace(item.Currency)
			runs[index].Items = append(runs[index].Items, item)
		}
		itemErr = itemRows.Err()
		itemRows.Close()
		if itemErr != nil {
			return nil, itemErr
		}
	}
	return runs, nil
}

func listTransactions(ctx context.Context, tx pgx.Tx, tenantID string) ([]finance.Transaction, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,idempotency_key,transaction_type,reference_type,reference_id,description,COALESCE(reverses_transaction_id::text,''),created_by,metadata,created_at FROM ledger_transactions WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC LIMIT 100`, tenantID)
	if err != nil {
		return nil, err
	}
	items := []finance.Transaction{}
	for rows.Next() {
		var item finance.Transaction
		var metadata []byte
		if err = rows.Scan(&item.ID, &item.TenantID, &item.IdempotencyKey, &item.TransactionType, &item.ReferenceType, &item.ReferenceID, &item.Description, &item.ReversesTransactionID, &item.CreatedBy, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range items {
		items[index].Entries = []finance.Entry{}
		entryRows, entryErr := tx.Query(ctx, `SELECT id,tenant_id,transaction_id,account_id,direction,currency,amount_minor,created_at FROM ledger_entries WHERE tenant_id=$1 AND transaction_id=$2 ORDER BY created_at,id`, tenantID, items[index].ID)
		if entryErr != nil {
			return nil, entryErr
		}
		for entryRows.Next() {
			var entry finance.Entry
			if entryErr = entryRows.Scan(&entry.ID, &entry.TenantID, &entry.TransactionID, &entry.AccountID, &entry.Direction, &entry.Currency, &entry.AmountMinor, &entry.CreatedAt); entryErr != nil {
				entryRows.Close()
				return nil, entryErr
			}
			entry.Currency = strings.TrimSpace(entry.Currency)
			items[index].Entries = append(items[index].Entries, entry)
		}
		entryErr = entryRows.Err()
		entryRows.Close()
		if entryErr != nil {
			return nil, entryErr
		}
	}
	return items, nil
}

func (s *Store) ListFinance(ctx context.Context, scope tenancy.Scope) (finance.Overview, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (finance.Overview, error) {
		result := finance.Overview{}
		rows, err := tx.Query(ctx, walletQuery()+` WHERE wallet.tenant_id=$1 ORDER BY wallet.created_at,wallet.id`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Wallets, err = listRows(rows, scanWallet)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, `
			SELECT account.id,account.tenant_id,COALESCE(account.wallet_id::text,''),account.code,account.name,account.account_type,account.purpose,account.currency,
			       COALESCE(sum(CASE WHEN account.account_type IN ('asset','expense') THEN CASE entry.direction WHEN 'debit' THEN entry.amount_minor ELSE -entry.amount_minor END ELSE CASE entry.direction WHEN 'credit' THEN entry.amount_minor ELSE -entry.amount_minor END END),0)
			FROM ledger_accounts account LEFT JOIN ledger_entries entry ON entry.tenant_id=account.tenant_id AND entry.account_id=account.id
			WHERE account.tenant_id=$1 GROUP BY account.id ORDER BY account.code`, scope.TenantID)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		result.Accounts = []finance.Account{}
		for rows.Next() {
			var account finance.Account
			if err = rows.Scan(&account.ID, &account.TenantID, &account.WalletID, &account.Code, &account.Name, &account.AccountType, &account.Purpose, &account.Currency, &account.BalanceMinor); err != nil {
				return result, err
			}
			account.Currency = strings.TrimSpace(account.Currency)
			result.Accounts = append(result.Accounts, account)
		}
		if err = rows.Err(); err != nil {
			return result, err
		}
		rows.Close()
		result.Transactions, err = listTransactions(ctx, tx, scope.TenantID)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, holdSelect+` WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Holds, err = listRows(rows, scanHold)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,customer_charge_id,wallet_id,currency,amount_minor,reason,status,ledger_transaction_id,idempotency_key,created_by,created_at FROM refunds WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Refunds, err = listRows(rows, scanRefund)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,customer_charge_id,beneficiary_type,beneficiary_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at FROM commissions WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Commissions, err = listRows(rows, scanCommission)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,provider_cost_id,provider_id,currency,amount_minor,settled_minor,status,payable_account_id,ledger_transaction_id,idempotency_key,created_by,created_at,updated_at FROM provider_payables WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ProviderPayables, err = listRows(rows, scanProviderPayable)
		if err != nil {
			return result, err
		}
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,source_type,source_id,beneficiary_type,beneficiary_id,currency,amount_minor,status,ledger_transaction_id,idempotency_key,created_by,created_at FROM settlements WHERE tenant_id=$1 ORDER BY created_at DESC,id DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Settlements, err = listRows(rows, scanSettlement)
		if err != nil {
			return result, err
		}
		result.ReconciliationRuns, err = listReconciliationRuns(ctx, tx, scope.TenantID)
		return result, err
	})
}

var _ application.FinanceStore = (*Store)(nil)
