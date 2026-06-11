package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agentstore/internal/db"
	"agentstore/internal/models"
	"agentstore/internal/paymentregistry"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const maskedValue = "••••••"

func maskField(v string) string {
	if v != "" {
		return maskedValue
	}
	return ""
}

// keepExisting returns the existing value when the incoming value is empty or the
// masked sentinel, preserving sensitive fields on partial saves.
func keepExisting(incoming, existing string) string {
	if incoming == "" || incoming == maskedValue {
		return existing
	}
	return incoming
}

// PaymentConfigHandler handles admin GET/PUT for WeChat Pay and Alipay credentials.
type PaymentConfigHandler struct {
	db       *db.MongoDB
	registry *paymentregistry.Registry
}

func NewPaymentConfigHandler(database *db.MongoDB, registry *paymentregistry.Registry) *PaymentConfigHandler {
	return &PaymentConfigHandler{db: database, registry: registry}
}

// GetWechat handles GET /admin/payment-config/wechat
func (h *PaymentConfigHandler) GetWechat(w http.ResponseWriter, r *http.Request) {
	var doc models.PaymentProviderConfig
	err := h.db.PaymentConfigs().FindOne(r.Context(), bson.M{"key": paymentregistry.KeyWechatPay}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"key":        paymentregistry.KeyWechatPay,
			"configured": false,
			"enabled":    false,
		})
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"key":          doc.Key,
		"appId":        doc.AppID,
		"mchId":        doc.MchID,
		"apiV3Key":     maskField(doc.APIv3Key),
		"privateKey":   maskField(doc.PrivateKey),
		"certSerialNo": doc.CertSerialNo,
		"notifyUrl":    doc.NotifyURL,
		"enabled":      doc.Enabled,
		"configured":   doc.MchID != "",
	})
}

// UpdateWechat handles PUT /admin/payment-config/wechat
func (h *PaymentConfigHandler) UpdateWechat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID        string `json:"appId"`
		MchID        string `json:"mchId"`
		APIv3Key     string `json:"apiV3Key"`
		PrivateKey   string `json:"privateKey"`
		CertSerialNo string `json:"certSerialNo"`
		NotifyURL    string `json:"notifyUrl"`
		Enabled      bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var existing models.PaymentProviderConfig
	findErr := h.db.PaymentConfigs().FindOne(ctx, bson.M{"key": paymentregistry.KeyWechatPay}).Decode(&existing)
	if findErr != nil && !errors.Is(findErr, mongo.ErrNoDocuments) {
		respondWithError(w, http.StatusInternalServerError, "Failed to load existing config")
		return
	}

	now := time.Now()
	_, upsertErr := h.db.PaymentConfigs().UpdateOne(
		ctx,
		bson.M{"key": paymentregistry.KeyWechatPay},
		bson.M{
			"$set": bson.M{
				"appId":        req.AppID,
				"mchId":        req.MchID,
				"apiV3Key":     keepExisting(req.APIv3Key, existing.APIv3Key),
				"privateKey":   keepExisting(req.PrivateKey, existing.PrivateKey),
				"certSerialNo": req.CertSerialNo,
				"notifyUrl":    req.NotifyURL,
				"enabled":      req.Enabled,
				"updatedAt":    now,
			},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"key":       paymentregistry.KeyWechatPay,
				"createdAt": now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if upsertErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	resp := map[string]string{"status": "saved"}
	if reloadErr := h.registry.Reload(ctx); reloadErr != nil {
		resp["warning"] = "Service initialization failed: " + reloadErr.Error()
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// GetAlipay handles GET /admin/payment-config/alipay
func (h *PaymentConfigHandler) GetAlipay(w http.ResponseWriter, r *http.Request) {
	var doc models.PaymentProviderConfig
	err := h.db.PaymentConfigs().FindOne(r.Context(), bson.M{"key": paymentregistry.KeyAlipay}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"key":        paymentregistry.KeyAlipay,
			"configured": false,
			"enabled":    false,
			"isSandbox":  true,
		})
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load config")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"key":        doc.Key,
		"appId":      doc.AppID,
		"privateKey": maskField(doc.PrivateKey),
		"publicKey":  maskField(doc.PublicKey),
		"notifyUrl":  doc.NotifyURL,
		"returnUrl":  doc.ReturnURL,
		"isSandbox":  doc.IsSandbox,
		"enabled":    doc.Enabled,
		"configured": doc.AppID != "",
	})
}

// UpdateAlipay handles PUT /admin/payment-config/alipay
func (h *PaymentConfigHandler) UpdateAlipay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AppID      string `json:"appId"`
		PrivateKey string `json:"privateKey"`
		PublicKey  string `json:"publicKey"`
		NotifyURL  string `json:"notifyUrl"`
		ReturnURL  string `json:"returnUrl"`
		IsSandbox  bool   `json:"isSandbox"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var existing models.PaymentProviderConfig
	findErr := h.db.PaymentConfigs().FindOne(ctx, bson.M{"key": paymentregistry.KeyAlipay}).Decode(&existing)
	if findErr != nil && !errors.Is(findErr, mongo.ErrNoDocuments) {
		respondWithError(w, http.StatusInternalServerError, "Failed to load existing config")
		return
	}

	now := time.Now()
	_, upsertErr := h.db.PaymentConfigs().UpdateOne(
		ctx,
		bson.M{"key": paymentregistry.KeyAlipay},
		bson.M{
			"$set": bson.M{
				"appId":      req.AppID,
				"privateKey": keepExisting(req.PrivateKey, existing.PrivateKey),
				"publicKey":  keepExisting(req.PublicKey, existing.PublicKey),
				"notifyUrl":  req.NotifyURL,
				"returnUrl":  req.ReturnURL,
				"isSandbox":  req.IsSandbox,
				"enabled":    req.Enabled,
				"updatedAt":  now,
			},
			"$setOnInsert": bson.M{
				"_id":       primitive.NewObjectID(),
				"key":       paymentregistry.KeyAlipay,
				"createdAt": now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if upsertErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	resp := map[string]string{"status": "saved"}
	if reloadErr := h.registry.Reload(ctx); reloadErr != nil {
		resp["warning"] = "Service initialization failed: " + reloadErr.Error()
	}
	respondWithJSON(w, http.StatusOK, resp)
}
