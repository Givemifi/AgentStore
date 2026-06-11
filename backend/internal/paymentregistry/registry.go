// Package paymentregistry manages live WeChat Pay and Alipay service instances.
// Call Reload to hot-swap credentials without restarting the server.
package paymentregistry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	alipayservice "agentstore/internal/alipay"
	"agentstore/internal/config"
	"agentstore/internal/db"
	"agentstore/internal/models"
	wechatservice "agentstore/internal/wechat"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	KeyWechatPay = "wechat_pay"
	KeyAlipay    = "alipay"
)

// Registry holds the live payment service instances and reloads them from DB on demand.
type Registry struct {
	mu     sync.RWMutex
	db     *db.MongoDB
	wechat *wechatservice.Service
	alipay *alipayservice.Service
}

// New creates a Registry backed by the given database.
func New(database *db.MongoDB) *Registry {
	return &Registry{db: database}
}

// Wechat returns the current WeChat Pay service; nil means not configured.
// Safe to call on a nil Registry.
func (r *Registry) Wechat() *wechatservice.Service {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.wechat
}

// Alipay returns the current Alipay service; nil means not configured.
// Safe to call on a nil Registry.
func (r *Registry) Alipay() *alipayservice.Service {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.alipay
}

// Reload reads both provider configs from DB and rebuilds the service instances.
// If either service fails to initialise, the old instances are preserved.
// On success, both instances are atomically replaced.
func (r *Registry) Reload(ctx context.Context) error {
	var wechatDoc, alipayDoc models.PaymentProviderConfig

	wechatErr := r.db.PaymentConfigs().FindOne(ctx, bson.M{"key": KeyWechatPay}).Decode(&wechatDoc)
	alipayErr := r.db.PaymentConfigs().FindOne(ctx, bson.M{"key": KeyAlipay}).Decode(&alipayDoc)

	if wechatErr != nil && !errors.Is(wechatErr, mongo.ErrNoDocuments) {
		return fmt.Errorf("paymentregistry: read wechat config: %w", wechatErr)
	}
	if alipayErr != nil && !errors.Is(alipayErr, mongo.ErrNoDocuments) {
		return fmt.Errorf("paymentregistry: read alipay config: %w", alipayErr)
	}

	// Build services outside the lock (wechat cert download is a network call).
	var newWechat *wechatservice.Service
	var newAlipay *alipayservice.Service

	if wechatErr == nil && wechatDoc.Enabled {
		svc, err := wechatservice.New(config.WeChatPayConfig{
			AppID:        wechatDoc.AppID,
			MchID:        wechatDoc.MchID,
			APIv3Key:     wechatDoc.APIv3Key,
			PrivateKey:   wechatDoc.PrivateKey,
			CertSerialNo: wechatDoc.CertSerialNo,
			NotifyURL:    wechatDoc.NotifyURL,
		})
		if err != nil {
			return fmt.Errorf("paymentregistry: init wechat: %w", err)
		}
		newWechat = svc
	}

	if alipayErr == nil && alipayDoc.Enabled {
		svc, err := alipayservice.New(config.AlipayConfig{
			AppID:      alipayDoc.AppID,
			PrivateKey: alipayDoc.PrivateKey,
			PublicKey:  alipayDoc.PublicKey,
			NotifyURL:  alipayDoc.NotifyURL,
			ReturnURL:  alipayDoc.ReturnURL,
			IsSandbox:  alipayDoc.IsSandbox,
		})
		if err != nil {
			return fmt.Errorf("paymentregistry: init alipay: %w", err)
		}
		newAlipay = svc
	}

	r.mu.Lock()
	r.wechat = newWechat
	r.alipay = newAlipay
	r.mu.Unlock()
	return nil
}

// WechatSeedConfig holds env/yaml WeChat credentials for one-time migration to DB.
type WechatSeedConfig struct {
	AppID        string
	MchID        string
	APIv3Key     string
	PrivateKey   string
	CertSerialNo string
	NotifyURL    string
}

// AlipaySeedConfig holds env/yaml Alipay credentials for one-time migration to DB.
type AlipaySeedConfig struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	NotifyURL  string
	ReturnURL  string
	IsSandbox  bool
}

// Seed writes env/yaml credentials to the DB if the collection is empty.
// This is a one-time migration; subsequent starts skip it.
func (r *Registry) Seed(ctx context.Context, wechat WechatSeedConfig, alipay AlipaySeedConfig) error {
	count, err := r.db.PaymentConfigs().CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("paymentregistry: seed count: %w", err)
	}
	if count > 0 {
		return nil // already populated
	}

	now := time.Now()

	if wechat.MchID != "" {
		doc := models.PaymentProviderConfig{
			ID:           primitive.NewObjectID(),
			Key:          KeyWechatPay,
			AppID:        wechat.AppID,
			MchID:        wechat.MchID,
			APIv3Key:     wechat.APIv3Key,
			PrivateKey:   wechat.PrivateKey,
			CertSerialNo: wechat.CertSerialNo,
			NotifyURL:    wechat.NotifyURL,
			Enabled:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if _, err := r.db.PaymentConfigs().InsertOne(ctx, doc); err != nil {
			return fmt.Errorf("paymentregistry: seed wechat: %w", err)
		}
	}

	if alipay.AppID != "" {
		doc := models.PaymentProviderConfig{
			ID:         primitive.NewObjectID(),
			Key:        KeyAlipay,
			AppID:      alipay.AppID,
			PrivateKey: alipay.PrivateKey,
			PublicKey:  alipay.PublicKey,
			NotifyURL:  alipay.NotifyURL,
			ReturnURL:  alipay.ReturnURL,
			IsSandbox:  alipay.IsSandbox,
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if _, err := r.db.PaymentConfigs().InsertOne(ctx, doc); err != nil {
			return fmt.Errorf("paymentregistry: seed alipay: %w", err)
		}
	}

	// Ensure unique index on key (idempotent).
	_, _ = r.db.PaymentConfigs().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return nil
}
