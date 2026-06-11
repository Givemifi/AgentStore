package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/events"
	"agentstore/internal/middleware"
	"agentstore/internal/models"
	"agentstore/internal/paymentregistry"
	stripeservice "agentstore/internal/stripe"
	"agentstore/internal/syslog"

	"go.mongodb.org/mongo-driver/bson"
	mongoDB "go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

// PaymentNotifyHandler handles async payment callbacks from WeChat Pay and Alipay,
// plus the frontend status-polling endpoint.
type PaymentNotifyHandler struct {
	registry *paymentregistry.Registry
	stripe   *stripeservice.Service // needed for NextInvoiceNumber
	db       *db.MongoDB
	events   events.Emitter
	syslog   *syslog.Logger
}

// NewPaymentNotifyHandler constructs a PaymentNotifyHandler.
func NewPaymentNotifyHandler(
	reg *paymentregistry.Registry,
	stripeSvc *stripeservice.Service,
	database *db.MongoDB,
	emitter events.Emitter,
	sysLogger *syslog.Logger,
) *PaymentNotifyHandler {
	return &PaymentNotifyHandler{
		registry: reg,
		stripe:   stripeSvc,
		db:       database,
		events:   emitter,
		syslog:   sysLogger,
	}
}

// HandleWechatNotify processes WeChat Pay V3 async payment notifications.
// POST /api/billing/wechat/notify — no auth, verified by WeChat platform signature.
func (h *PaymentNotifyHandler) HandleWechatNotify(w http.ResponseWriter, r *http.Request) {
	if h.registry.Wechat() == nil {
		http.Error(w, "WeChat Pay not configured", http.StatusServiceUnavailable)
		return
	}

	result, err := h.registry.Wechat().ParseAndVerifyNotify(r)
	if err != nil {
		slog.Error("WechatNotify: parse/verify failed", "error", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if result.TradeState != "SUCCESS" {
		slog.Info("WechatNotify: non-success state, ignoring", "state", result.TradeState, "outTradeNo", result.OutTradeNo)
		w.WriteHeader(http.StatusOK)
		return
	}

	var notifyFen int64
	if result.Amount != nil {
		notifyFen = int64(result.Amount.Total)
	}

	ctx := r.Context()
	if err := h.completeDomesticOrder(ctx, result.OutTradeNo, "wechat_h5", notifyFen); err != nil {
		slog.Error("WechatNotify: complete order failed", "outTradeNo", result.OutTradeNo, "error", err)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	// WeChat Pay V3 requires a JSON {"code":"SUCCESS"} response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":"SUCCESS","message":"成功"}`)) //nolint:errcheck
}

// HandleAlipayNotify processes Alipay async payment notifications.
// POST /api/billing/alipay/notify — no auth, verified by Alipay RSA2 signature.
func (h *PaymentNotifyHandler) HandleAlipayNotify(w http.ResponseWriter, r *http.Request) {
	if h.registry.Alipay() == nil {
		http.Error(w, "Alipay not configured", http.StatusServiceUnavailable)
		return
	}

	bm, err := h.registry.Alipay().ParseAndVerifyNotify(r)
	if err != nil {
		slog.Error("AlipayNotify: parse/verify failed", "error", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	tradeStatus := bm.GetString("trade_status")
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		slog.Info("AlipayNotify: non-success status, ignoring", "status", tradeStatus)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success")) //nolint:errcheck
		return
	}

	outTradeNo := bm.GetString("out_trade_no")

	// Convert total_amount (yuan string, e.g. "12.50") → fen for validation.
	var amountFen int64
	if f, err := parseYuanToFen(bm.GetString("total_amount")); err == nil {
		amountFen = f
	}

	ctx := r.Context()
	if err := h.completeDomesticOrder(ctx, outTradeNo, "alipay", amountFen); err != nil {
		slog.Error("AlipayNotify: complete order failed", "outTradeNo", outTradeNo, "error", err)
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}

	// Alipay requires plain-text "success" response.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success")) //nolint:errcheck
}

// GetPaymentStatus polls the status of a domestic payment order.
// GET /api/billing/payment/status?outTradeNo=xxx — requires JWT + tenant.
func (h *PaymentNotifyHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	outTradeNo := r.URL.Query().Get("outTradeNo")
	if outTradeNo == "" {
		respondWithError(w, http.StatusBadRequest, "outTradeNo is required")
		return
	}

	var order models.PaymentOrder
	if err := h.db.PaymentOrders().FindOne(ctx, bson.M{"outTradeNo": outTradeNo}).Decode(&order); err != nil {
		respondWithError(w, http.StatusNotFound, "Order not found")
		return
	}

	// Verify order belongs to the authenticated tenant (prevent cross-tenant info leak).
	if tenant, ok := middleware.GetTenantFromContext(ctx); ok {
		if order.TenantID != tenant.ID {
			respondWithError(w, http.StatusForbidden, "Order not found")
			return
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":     order.Status,
		"provider":   order.Provider,
		"credits":    order.Credits,
		"outTradeNo": order.OutTradeNo,
	})
}

// completeDomesticOrder atomically transitions a payment_order from pending→completed,
// adds credits to the tenant, and records a FinancialTransaction.
// Idempotent: duplicate notifications for the same outTradeNo are silently skipped.
func (h *PaymentNotifyHandler) completeDomesticOrder(ctx context.Context, outTradeNo, provider string, notifyAmountFen int64) error {
	now := time.Now()

	// Atomic transition: only matches if status == "pending".
	result := h.db.PaymentOrders().FindOneAndUpdate(ctx,
		bson.M{"outTradeNo": outTradeNo, "status": "pending"},
		bson.M{"$set": bson.M{
			"status":      "completed",
			"completedAt": now,
		}},
		mopts.FindOneAndUpdate().SetReturnDocument(mopts.After),
	)
	if result.Err() == mongoDB.ErrNoDocuments {
		// Either order doesn't exist or already completed — safe to ignore.
		slog.Info("DomesticPay: duplicate notify or already completed, skipping", "outTradeNo", outTradeNo)
		return nil
	}
	if result.Err() != nil {
		return fmt.Errorf("complete order update: %w", result.Err())
	}

	var order models.PaymentOrder
	if err := result.Decode(&order); err != nil {
		return fmt.Errorf("decode completed order: %w", err)
	}

	// Security: amount in notify must match recorded amount (±1 fen for floating-point rounding).
	if notifyAmountFen > 0 && abs64(notifyAmountFen-order.AmountCents) > 1 {
		h.syslog.Critical(ctx, fmt.Sprintf(
			"SECURITY: domestic pay amount mismatch — order %s recorded %d fen, notify %d fen",
			outTradeNo, order.AmountCents, notifyAmountFen,
		))
		return fmt.Errorf("amount mismatch: order=%d fen, notify=%d fen", order.AmountCents, notifyAmountFen)
	}

	// Credit tenant.
	if _, err := h.db.Tenants().UpdateOne(ctx,
		bson.M{"_id": order.TenantID},
		bson.M{
			"$inc": bson.M{"purchasedCredits": order.Credits},
			"$set": bson.M{"updatedAt": now},
		},
	); err != nil {
		return fmt.Errorf("add credits: %w", err)
	}

	// Record completed transaction.
	h.recordDomesticTransaction(ctx, order, provider)

	h.syslog.High(ctx, fmt.Sprintf(
		"DomesticPay: order %s completed — tenant %s, %d credits, ¥%.2f (%s)",
		outTradeNo, order.TenantID.Hex(), order.Credits, float64(order.AmountCents)/100.0, provider,
	))

	h.events.Emit(events.Event{
		Type:      events.EventCreditsPurchased,
		Timestamp: now,
		Data: map[string]interface{}{
			"tenantId":    order.TenantID.Hex(),
			"bundleId":    order.BundleID.Hex(),
			"bundleName":  order.BundleName,
			"credits":     order.Credits,
			"amountCents": order.AmountCents,
			"currency":    order.Currency,
			"provider":    provider,
		},
	})

	return nil
}

// recordDomesticTransaction writes a FinancialTransaction for a completed domestic order.
func (h *PaymentNotifyHandler) recordDomesticTransaction(ctx context.Context, order models.PaymentOrder, provider string) {
	var invoiceNum string
	if h.stripe != nil {
		var err error
		invoiceNum, err = h.stripe.NextInvoiceNumber(ctx)
		if err != nil {
			slog.Error("DomesticPay: failed to generate invoice number", "error", err)
		}
	}
	if invoiceNum == "" {
		rb := make([]byte, 4)
		rand.Read(rb) //nolint:errcheck
		invoiceNum = fmt.Sprintf("INV-ERR-%d-%s", time.Now().UnixNano(), hex.EncodeToString(rb))
	}

	bundleID := order.BundleID
	tx := models.FinancialTransaction{
		TenantID:        order.TenantID,
		UserID:          order.UserID,
		Type:            models.TransactionCreditPurchase,
		AmountCents:     order.AmountCents,
		SubtotalCents:   order.AmountCents,
		TaxAmountCents:  0,
		Currency:        order.Currency,
		Description:     order.BundleName,
		InvoiceNumber:   invoiceNum,
		PaymentProvider: provider,
		OutTradeNo:      order.OutTradeNo,
		BundleID:        &bundleID,
		BundleName:      order.BundleName,
		CreatedAt:       time.Now(),
	}

	if _, err := h.db.FinancialTransactions().InsertOne(ctx, tx); err != nil {
		slog.Error("DomesticPay: failed to record transaction", "error", err)
	}
}

// abs64 returns the absolute value of a.
func abs64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

// parseYuanToFen converts a CNY yuan string (e.g. "12.50") to fen (1250).
func parseYuanToFen(yuan string) (int64, error) {
	if yuan == "" {
		return 0, fmt.Errorf("empty amount")
	}
	var val float64
	if _, err := fmt.Sscanf(yuan, "%f", &val); err != nil {
		return 0, err
	}
	return int64(val * 100), nil
}
