package finance

import "time"

type Wallet struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	OwnerType      string    `json:"owner_type"`
	OwnerID        string    `json:"owner_id"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	AvailableMinor int64     `json:"available_minor"`
	HeldMinor      int64     `json:"held_minor"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Account struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	WalletID     string    `json:"wallet_id,omitempty"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	AccountType  string    `json:"account_type"`
	Purpose      string    `json:"purpose"`
	Currency     string    `json:"currency"`
	BalanceMinor int64     `json:"balance_minor"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

type Entry struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Direction     string    `json:"direction"`
	Currency      string    `json:"currency"`
	AmountMinor   int64     `json:"amount_minor"`
	CreatedAt     time.Time `json:"created_at"`
}

type Transaction struct {
	ID                    string         `json:"id"`
	TenantID              string         `json:"tenant_id"`
	IdempotencyKey        string         `json:"idempotency_key"`
	TransactionType       string         `json:"transaction_type"`
	ReferenceType         string         `json:"reference_type"`
	ReferenceID           string         `json:"reference_id"`
	Description           string         `json:"description"`
	ReversesTransactionID string         `json:"reverses_transaction_id,omitempty"`
	CreatedBy             string         `json:"created_by"`
	Metadata              map[string]any `json:"metadata"`
	Entries               []Entry        `json:"entries"`
	CreatedAt             time.Time      `json:"created_at"`
}

type Hold struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	WalletID            string    `json:"wallet_id"`
	OrderID             string    `json:"order_id"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	CapturedMinor       int64     `json:"captured_minor"`
	ReleasedMinor       int64     `json:"released_minor"`
	RemainingMinor      int64     `json:"remaining_minor"`
	Status              string    `json:"status"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Adjustment struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	WalletID            string    `json:"wallet_id"`
	Direction           string    `json:"direction"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	Reason              string    `json:"reason"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}

type Refund struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	CustomerChargeID    string    `json:"customer_charge_id"`
	WalletID            string    `json:"wallet_id"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	Reason              string    `json:"reason"`
	Status              string    `json:"status"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}

type Commission struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	CustomerChargeID    string    `json:"customer_charge_id"`
	BeneficiaryType     string    `json:"beneficiary_type"`
	BeneficiaryID       string    `json:"beneficiary_id"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	SettledMinor        int64     `json:"settled_minor"`
	Status              string    `json:"status"`
	PayableAccountID    string    `json:"payable_account_id"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProviderPayable struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	ProviderCostID      string    `json:"provider_cost_id"`
	ProviderID          string    `json:"provider_id"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	SettledMinor        int64     `json:"settled_minor"`
	Status              string    `json:"status"`
	PayableAccountID    string    `json:"payable_account_id"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Settlement struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	SourceType          string    `json:"source_type"`
	SourceID            string    `json:"source_id"`
	BeneficiaryType     string    `json:"beneficiary_type"`
	BeneficiaryID       string    `json:"beneficiary_id"`
	Currency            string    `json:"currency"`
	AmountMinor         int64     `json:"amount_minor"`
	Status              string    `json:"status"`
	LedgerTransactionID string    `json:"ledger_transaction_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}

type ReconciliationItem struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	RunID         string    `json:"run_id"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   string    `json:"reference_id"`
	Currency      string    `json:"currency"`
	ExpectedMinor int64     `json:"expected_minor"`
	ActualMinor   int64     `json:"actual_minor"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReconciliationRun struct {
	ID               string               `json:"id"`
	TenantID         string               `json:"tenant_id"`
	OrderID          string               `json:"order_id,omitempty"`
	Status           string               `json:"status"`
	CheckedCount     int                  `json:"checked_count"`
	DiscrepancyCount int                  `json:"discrepancy_count"`
	CreatedBy        string               `json:"created_by"`
	IdempotencyKey   string               `json:"idempotency_key"`
	StartedAt        time.Time            `json:"started_at"`
	CompletedAt      time.Time            `json:"completed_at"`
	Items            []ReconciliationItem `json:"items"`
}

type Overview struct {
	Wallets            []Wallet            `json:"wallets"`
	Accounts           []Account           `json:"accounts"`
	Transactions       []Transaction       `json:"transactions"`
	Holds              []Hold              `json:"holds"`
	Refunds            []Refund            `json:"refunds"`
	Commissions        []Commission        `json:"commissions"`
	ProviderPayables   []ProviderPayable   `json:"provider_payables"`
	Settlements        []Settlement        `json:"settlements"`
	ReconciliationRuns []ReconciliationRun `json:"reconciliation_runs"`
}

type EntryInput struct {
	AccountID   string
	Direction   string
	AmountMinor int64
}
