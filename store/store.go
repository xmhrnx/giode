// Package store sells in-app products and subscriptions: Google Play
// Billing on Android, the App Store (StoreKit) on iOS and macOS.
// Elsewhere its functions return giode.ErrNotSupported.
//
// Products are set up in the Play Console or App Store Connect; their
// ids are what Products takes. A purchase flow is:
//
//	ps, err := store.Products(ctx, "pro", "coins_100")
//	p, err := store.Buy(ctx, app, ps[0])
//	// Check p.Receipt on the server, then grant the product and:
//	err = store.Finish(ctx, p, false) // true for consumables
//
// A purchase not finished within three days is refunded by Google
// Play, and redelivered by the App Store on every launch, through
// OnUpdate.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/xmhrnx/giode"
)

// ErrCanceled is returned by Buy when the user cancels it.
var ErrCanceled = errors.New("store: purchase canceled")

// ErrOwned is returned by Buy when the user already owns the
// product: call Owned and finish or restore it.
var ErrOwned = errors.New("store: product already owned")

// Product is a product for sale.
type Product struct {
	ID          string
	Title       string
	Description string
	// Price is formatted for the user's store country ("€4,99").
	Price string
	// PriceMicros is the price in millionths of Currency's unit.
	PriceMicros int64
	// Currency is the ISO 4217 code ("EUR").
	Currency string
	// Subscription reports a subscription; Period is its billing period
	// as ISO 8601 ("P1M" monthly, "P1Y" yearly).
	Subscription bool
	Period       string
}

// Purchase is a completed (or pending) purchase.
type Purchase struct {
	// ID is the order id (Google Play) or transaction id (App Store).
	ID        string
	ProductID string
	// Token identifies the purchase to the store's server API: Google
	// Play's purchase token, the App Store transaction id.
	Token string
	Time  time.Time
	// Pending purchases wait for payment (cash, a parent's approval);
	// they arrive through OnUpdate once paid. Don't grant them yet.
	Pending bool
	// Acknowledged reports a Google Play purchase already finished.
	Acknowledged bool
	// Receipt is what the server verifies: the purchase JSON (Google
	// Play, with Signature), or the app's base64 App Store receipt.
	Receipt   string
	Signature string
}

// Products returns the products with the given ids; ids the store
// doesn't know are left out.
func Products(ctx context.Context, ids ...string) ([]Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return products(ctx, ids)
}

// Buy buys p, showing the store's payment sheet. It returns
// ErrCanceled when the user cancels it.
func Buy(ctx context.Context, a *giode.App, p Product) (Purchase, error) {
	return purchase(ctx, a, p)
}

// Finish completes a purchase once the app granted it: Google Play
// acknowledges it (or consumes it, when consume is set, so it can be
// bought again), the App Store stops redelivering it.
func Finish(ctx context.Context, p Purchase, consume bool) error {
	return finish(ctx, p, consume)
}

// Owned returns the user's purchases: products not consumed and active
// subscriptions on Google Play; on the App Store, the purchases it
// restores (non-consumables and subscriptions), which may ask the user
// to sign in.
func Owned(ctx context.Context) ([]Purchase, error) {
	return owned(ctx)
}

var (
	updateMu sync.Mutex
	onUpdate func(Purchase)
	updates  []Purchase
)

// OnUpdate sets the function receiving purchases completed outside a
// Buy call: pending ones paid later, interrupted ones, App Store
// renewals and purchases not finished before. It runs on its own
// goroutine. Updates from before the call are delivered to it.
func OnUpdate(fn func(Purchase)) {
	updateMu.Lock()
	onUpdate = fn
	q := updates
	updates = nil
	updateMu.Unlock()
	if fn != nil {
		for _, p := range q {
			go fn(p)
		}
	}
	startUpdates()
}

func update(p Purchase) {
	updateMu.Lock()
	fn := onUpdate
	if fn == nil {
		updates = append(updates, p)
	}
	updateMu.Unlock()
	if fn != nil {
		go fn(p)
	}
}

// The native side reports in JSON.

type productJSON struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Price        string `json:"price"`
	Micros       int64  `json:"micros"`
	Currency     string `json:"currency"`
	Subscription bool   `json:"subscription"`
	Period       string `json:"period"`
}

type purchaseJSON struct {
	ID           string `json:"id"`
	Product      string `json:"product"`
	Token        string `json:"token"`
	Time         int64  `json:"time"` // ms since the epoch
	Pending      bool   `json:"pending"`
	Acknowledged bool   `json:"acknowledged"`
	Receipt      string `json:"receipt"`
	Signature    string `json:"signature"`
}

func parseProducts(s string) ([]Product, error) {
	var ps []productJSON
	if err := json.Unmarshal([]byte(s), &ps); err != nil {
		return nil, err
	}
	out := make([]Product, len(ps))
	for i, p := range ps {
		out[i] = Product{ID: p.ID, Title: p.Title, Description: p.Description, Price: p.Price,
			PriceMicros: p.Micros, Currency: p.Currency, Subscription: p.Subscription, Period: p.Period}
	}
	return out, nil
}

func (p purchaseJSON) purchase() Purchase {
	return Purchase{ID: p.ID, ProductID: p.Product, Token: p.Token, Time: time.UnixMilli(p.Time),
		Pending: p.Pending, Acknowledged: p.Acknowledged, Receipt: p.Receipt, Signature: p.Signature}
}

func parsePurchase(s string) (Purchase, error) {
	var p purchaseJSON
	err := json.Unmarshal([]byte(s), &p)
	return p.purchase(), err
}

func parsePurchases(s string) ([]Purchase, error) {
	var ps []purchaseJSON
	if err := json.Unmarshal([]byte(s), &ps); err != nil {
		return nil, err
	}
	out := make([]Purchase, len(ps))
	for i, p := range ps {
		out[i] = p.purchase()
	}
	return out, nil
}

// result is a native answer: JSON, or an error message ("canceled" and
// "owned" being ErrCanceled and ErrOwned).
type result struct {
	json string
	err  error
}

func nativeError(msg string) error {
	switch msg {
	case "":
		return nil
	case "canceled":
		return ErrCanceled
	case "owned":
		return ErrOwned
	}
	return errors.New("store: " + msg)
}
