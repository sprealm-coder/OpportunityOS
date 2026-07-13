package platform

import "fmt"

type Money struct {
	Currency string `json:"currency"`
	Minor    int64  `json:"minor"`
}

func NewMoney(currency string, minor int64) (Money, error) {
	if len(currency) != 3 {
		return Money{}, fmt.Errorf("currency must be an ISO 4217 code")
	}
	return Money{Currency: currency, Minor: minor}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s != %s", m.Currency, other.Currency)
	}
	return Money{Currency: m.Currency, Minor: m.Minor + other.Minor}, nil
}
