package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Revenue   AccountType = "revenue"
	Expense   AccountType = "expense"
	Equity    AccountType = "equity"
)

type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

type Account struct {
	ID, TenantID, Code, Name, Currency string
	Type                               AccountType
}
type Entry struct {
	ID, TenantID, TransactionID, AccountID, Currency string
	Direction                                        Direction
	AmountMinor                                      int64
	CreatedAt                                        time.Time
}
type Transaction struct {
	ID, TenantID, IdempotencyKey, ReferenceType, ReferenceID, Description string
	Entries                                                               []Entry
	CreatedAt                                                             time.Time
	ReversesTransactionID                                                 string
}

type Ledger struct {
	mu           sync.RWMutex
	accounts     map[string]Account
	transactions map[string]Transaction
	idempotency  map[string]string
}

func New() *Ledger {
	return &Ledger{accounts: map[string]Account{}, transactions: map[string]Transaction{}, idempotency: map[string]string{}}
}
func key(tenantID, id string) string { return tenantID + "/" + id }

func (l *Ledger) AddAccount(account Account) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if account.TenantID == "" {
		return platform.ErrTenantRequired
	}
	if len(account.Currency) != 3 {
		return fmt.Errorf("invalid account currency")
	}
	if account.ID == "" {
		account.ID = platform.NewID("acct")
	}
	k := key(account.TenantID, account.ID)
	if _, exists := l.accounts[k]; exists {
		return fmt.Errorf("account exists")
	}
	l.accounts[k] = account
	return nil
}

func (l *Ledger) Post(transaction Transaction) (Transaction, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if transaction.TenantID == "" {
		return Transaction{}, platform.ErrTenantRequired
	}
	if transaction.IdempotencyKey == "" {
		return Transaction{}, platform.ErrIdempotencyKeyRequired
	}
	idemKey := key(transaction.TenantID, transaction.IdempotencyKey)
	if id, ok := l.idempotency[idemKey]; ok {
		return l.transactions[key(transaction.TenantID, id)], nil
	}
	if len(transaction.Entries) < 2 {
		return Transaction{}, fmt.Errorf("transaction requires at least two entries")
	}
	var debits, credits int64
	currency := ""
	for i := range transaction.Entries {
		entry := &transaction.Entries[i]
		account, ok := l.accounts[key(transaction.TenantID, entry.AccountID)]
		if !ok {
			return Transaction{}, fmt.Errorf("account %s not found in tenant", entry.AccountID)
		}
		if entry.AmountMinor <= 0 {
			return Transaction{}, fmt.Errorf("entry amount must be positive")
		}
		if currency == "" {
			currency = entry.Currency
		}
		if entry.Currency != currency || entry.Currency != account.Currency {
			return Transaction{}, fmt.Errorf("transaction currency mismatch")
		}
		switch entry.Direction {
		case Debit:
			debits += entry.AmountMinor
		case Credit:
			credits += entry.AmountMinor
		default:
			return Transaction{}, fmt.Errorf("invalid entry direction")
		}
	}
	if debits != credits {
		return Transaction{}, fmt.Errorf("unbalanced transaction: debits=%d credits=%d", debits, credits)
	}
	transaction.ID = platform.NewID("txn")
	transaction.CreatedAt = time.Now().UTC()
	for i := range transaction.Entries {
		transaction.Entries[i].ID = platform.NewID("entry")
		transaction.Entries[i].TenantID = transaction.TenantID
		transaction.Entries[i].TransactionID = transaction.ID
		transaction.Entries[i].CreatedAt = transaction.CreatedAt
	}
	l.transactions[key(transaction.TenantID, transaction.ID)] = transaction
	l.idempotency[idemKey] = transaction.ID
	return transaction, nil
}

func (l *Ledger) Balance(tenantID, accountID string) (int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	account, ok := l.accounts[key(tenantID, accountID)]
	if !ok {
		return 0, fmt.Errorf("account not found")
	}
	var balance int64
	for _, txn := range l.transactions {
		if txn.TenantID != tenantID {
			continue
		}
		for _, entry := range txn.Entries {
			if entry.AccountID != accountID {
				continue
			}
			normalDebit := account.Type == Asset || account.Type == Expense
			if (entry.Direction == Debit) == normalDebit {
				balance += entry.AmountMinor
			} else {
				balance -= entry.AmountMinor
			}
		}
	}
	return balance, nil
}

func (l *Ledger) Reverse(tenantID, transactionID, idempotencyKey, description string) (Transaction, error) {
	l.mu.RLock()
	original, ok := l.transactions[key(tenantID, transactionID)]
	l.mu.RUnlock()
	if !ok {
		return Transaction{}, fmt.Errorf("transaction not found")
	}
	entries := make([]Entry, len(original.Entries))
	for i, e := range original.Entries {
		direction := Debit
		if e.Direction == Debit {
			direction = Credit
		}
		entries[i] = Entry{AccountID: e.AccountID, Currency: e.Currency, Direction: direction, AmountMinor: e.AmountMinor}
	}
	return l.Post(Transaction{TenantID: tenantID, IdempotencyKey: idempotencyKey, ReferenceType: "reversal", ReferenceID: transactionID, Description: description, Entries: entries, ReversesTransactionID: transactionID})
}
