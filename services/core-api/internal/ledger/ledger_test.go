package ledger

import "testing"

func TestLedgerBalanceIdempotencyAndReversal(t *testing.T) {
	l := New()
	for _, account := range []Account{{ID: "cash", TenantID: "tenant", Code: "cash", Name: "Cash", Currency: "USD", Type: Asset}, {ID: "revenue", TenantID: "tenant", Code: "revenue", Name: "Revenue", Currency: "USD", Type: Revenue}} {
		if err := l.AddAccount(account); err != nil {
			t.Fatal(err)
		}
	}
	txn := Transaction{TenantID: "tenant", IdempotencyKey: "charge-1", ReferenceType: "order", ReferenceID: "order-1", Entries: []Entry{{AccountID: "cash", Currency: "USD", Direction: Debit, AmountMinor: 1000}, {AccountID: "revenue", Currency: "USD", Direction: Credit, AmountMinor: 1000}}}
	posted, err := l.Post(txn)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := l.Post(txn)
	if err != nil || duplicate.ID != posted.ID {
		t.Fatal("idempotency failed")
	}
	cash, err := l.Balance("tenant", "cash")
	if err != nil || cash != 1000 {
		t.Fatalf("cash=%d err=%v", cash, err)
	}
	if _, err := l.Reverse("tenant", posted.ID, "reverse-1", "test reversal"); err != nil {
		t.Fatal(err)
	}
	cash, err = l.Balance("tenant", "cash")
	if err != nil || cash != 0 {
		t.Fatalf("cash after reversal=%d err=%v", cash, err)
	}
}
func TestLedgerRejectsUnbalancedAndCrossTenant(t *testing.T) {
	l := New()
	_ = l.AddAccount(Account{ID: "cash", TenantID: "a", Currency: "USD", Type: Asset})
	_ = l.AddAccount(Account{ID: "revenue", TenantID: "b", Currency: "USD", Type: Revenue})
	_, err := l.Post(Transaction{TenantID: "a", IdempotencyKey: "x", Entries: []Entry{{AccountID: "cash", Currency: "USD", Direction: Debit, AmountMinor: 100}, {AccountID: "revenue", Currency: "USD", Direction: Credit, AmountMinor: 90}}})
	if err == nil {
		t.Fatal("expected cross-tenant or balance error")
	}
}
