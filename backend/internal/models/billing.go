package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BillingStatus represents the current billing state of a tenant.
type BillingStatus string

const (
	BillingStatusNone     BillingStatus = "none"
	BillingStatusActive   BillingStatus = "active"
	BillingStatusPastDue  BillingStatus = "past_due"
	BillingStatusCanceled BillingStatus = "canceled"
)

// TransactionType categorizes financial transactions.
type TransactionType string

const (
	TransactionSubscription   TransactionType = "subscription"
	TransactionCreditPurchase TransactionType = "credit_purchase"
	TransactionRefund         TransactionType = "refund"
)

// FinancialTransaction records every payment event.
type FinancialTransaction struct {
	ID                   primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID             primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	UserID               primitive.ObjectID  `json:"userId" bson:"userId" validate:"required"`
	Type                 TransactionType     `json:"type" bson:"type" validate:"required,oneof=subscription credit_purchase refund"`
	AmountCents          int64               `json:"amountCents" bson:"amountCents"`
	SubtotalCents        int64               `json:"subtotalCents" bson:"subtotalCents"`
	TaxAmountCents       int64               `json:"taxAmountCents" bson:"taxAmountCents"`
	Currency             string              `json:"currency" bson:"currency" validate:"required,len=3"`
	Description          string              `json:"description" bson:"description"`
	InvoiceNumber        string              `json:"invoiceNumber" bson:"invoiceNumber" validate:"required"`
	StripeSessionID      string              `json:"stripeSessionId,omitempty" bson:"stripeSessionId,omitempty"`
	StripeInvoiceID      string              `json:"stripeInvoiceId,omitempty" bson:"stripeInvoiceId,omitempty"`
	StripeSubscriptionID string              `json:"stripeSubscriptionId,omitempty" bson:"stripeSubscriptionId,omitempty"`
	PaymentProvider      string              `json:"paymentProvider,omitempty" bson:"paymentProvider,omitempty"` // "stripe"|"wechat_h5"|"alipay"
	OutTradeNo           string              `json:"outTradeNo,omitempty" bson:"outTradeNo,omitempty"`           // WeChat/Alipay order ID
	PlanID               *primitive.ObjectID `json:"planId,omitempty" bson:"planId,omitempty"`
	PlanName             string              `json:"planName,omitempty" bson:"planName,omitempty"`
	BundleID             *primitive.ObjectID `json:"bundleId,omitempty" bson:"bundleId,omitempty"`
	BundleName           string              `json:"bundleName,omitempty" bson:"bundleName,omitempty"`
	BillingInterval      string              `json:"billingInterval,omitempty" bson:"billingInterval,omitempty"`
	CreatedAt            time.Time           `json:"createdAt" bson:"createdAt" validate:"required"`
}

// PaymentOrder tracks a pending domestic-payment order (WeChat Pay / Alipay)
// before the async notify confirms payment. One record per order attempt.
type PaymentOrder struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OutTradeNo  string             `json:"outTradeNo" bson:"outTradeNo"`     // unique, merchant-side order ID
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`
	BundleID    primitive.ObjectID `json:"bundleId" bson:"bundleId"`
	BundleName  string             `json:"bundleName" bson:"bundleName"`
	Credits     int64              `json:"credits" bson:"credits"`
	AmountCents int64              `json:"amountCents" bson:"amountCents"` // CNY fen
	AmountUSD   int64              `json:"amountUSD" bson:"amountUSD"`     // original USD cents
	Currency    string             `json:"currency" bson:"currency"`       // "cny"
	Provider    string             `json:"provider" bson:"provider"`       // "wechat_h5" | "alipay"
	// Status: "pending" → "completed" | "failed"
	Status      string             `json:"status" bson:"status"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
}

// StripeMapping maps internal entities (plans, bundles) to Stripe Products/Prices.
type StripeMapping struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	EntityType      string             `bson:"entityType"`
	EntityID        primitive.ObjectID `bson:"entityId"`
	StripePriceID   string             `bson:"stripePriceId"`
	StripeProductID string             `bson:"stripeProductId"`
	CreatedAt       time.Time          `bson:"createdAt"`
}

// InvoiceCounter is used for atomic invoice number generation.
type InvoiceCounter struct {
	ID    string `bson:"_id"`
	Value int64  `bson:"value"`
}

// PaymentProviderConfig stores WeChat Pay or Alipay credentials in MongoDB.
// Key distinguishes providers: "wechat_pay" or "alipay".
type PaymentProviderConfig struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"   json:"id,omitempty"`
	Key          string             `bson:"key"             json:"key"          validate:"required,oneof=wechat_pay alipay"`
	AppID        string             `bson:"appId"           json:"appId"`
	// WeChat-specific
	MchID        string             `bson:"mchId"           json:"mchId"`
	APIv3Key     string             `bson:"apiV3Key"        json:"apiV3Key"`
	CertSerialNo string             `bson:"certSerialNo"    json:"certSerialNo"`
	// Alipay-specific
	PublicKey    string             `bson:"publicKey"       json:"publicKey"`
	ReturnURL    string             `bson:"returnUrl"       json:"returnUrl"`
	IsSandbox    bool               `bson:"isSandbox"       json:"isSandbox"`
	// Shared
	PrivateKey   string             `bson:"privateKey"      json:"privateKey"`
	NotifyURL    string             `bson:"notifyUrl"       json:"notifyUrl"`
	Enabled      bool               `bson:"enabled"         json:"enabled"`
	CreatedAt    time.Time          `bson:"createdAt"       json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"       json:"updatedAt"`
}

// DailyMetric stores daily business metrics for dashboard charts.
type DailyMetric struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Date      string             `json:"date" bson:"date"`
	DAU       int64              `json:"dau" bson:"dau"`
	WAU       int64              `json:"wau" bson:"wau"`
	MAU       int64              `json:"mau" bson:"mau"`
	Revenue   int64              `json:"revenue" bson:"revenue"`
	ARR       int64              `json:"arr" bson:"arr"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}
