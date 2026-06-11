// Package wechat wraps the WeChat Pay V3 API for H5 payment (non-WeChat browser).
package wechat

import (
	"context"
	"fmt"
	"net/http"

	"agentstore/internal/config"

	"github.com/go-pay/gopay"
	wechatV3 "github.com/go-pay/gopay/wechat/v3"
)

// Service is a thin wrapper around the gopay WeChat V3 client.
type Service struct {
	client *wechatV3.ClientV3
	cfg    config.WeChatPayConfig
}

// New initialises the WeChat Pay V3 client.
// Returns nil, nil when the configuration is empty — WeChat Pay is treated as
// an optional feature; the caller should check for nil before using the service.
func New(cfg config.WeChatPayConfig) (*Service, error) {
	if cfg.MchID == "" || cfg.APIv3Key == "" || cfg.PrivateKey == "" || cfg.CertSerialNo == "" {
		return nil, nil
	}

	client, err := wechatV3.NewClientV3(cfg.MchID, cfg.CertSerialNo, cfg.APIv3Key, cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("wechat: init client: %w", err)
	}

	// Download and cache platform certificates for outbound signature verification.
	if err = client.AutoVerifySign(); err != nil {
		return nil, fmt.Errorf("wechat: load platform certs: %w", err)
	}

	return &Service{client: client, cfg: cfg}, nil
}

// H5OrderResult is returned by CreateH5Order.
type H5OrderResult struct {
	H5URL string // redirect URL that triggers WeChat Pay on the device
}

// CreateH5Order places an H5 payment order and returns the redirect URL.
//   - outTradeNo: unique merchant order ID (≤32 chars, [A-Za-z0-9_\-])
//   - description: product description shown to the payer
//   - clientIP: payer's IP address (required by WeChat API)
//   - amountFen: amount in CNY 分 (e.g. 100 = ¥1.00)
func (s *Service) CreateH5Order(ctx context.Context, outTradeNo, description, clientIP string, amountFen int64) (*H5OrderResult, error) {
	if s.cfg.NotifyURL == "" {
		return nil, fmt.Errorf("wechat: notify_url not configured")
	}

	bm := make(gopay.BodyMap)
	bm.Set("appid", s.cfg.AppID)
	bm.Set("description", description)
	bm.Set("out_trade_no", outTradeNo)
	bm.Set("notify_url", s.cfg.NotifyURL)
	bm.SetBodyMap("amount", func(b gopay.BodyMap) {
		b.Set("total", amountFen)
		b.Set("currency", "CNY")
	})
	bm.SetBodyMap("scene_info", func(b gopay.BodyMap) {
		b.Set("payer_client_ip", clientIP)
		b.SetBodyMap("h5_info", func(h gopay.BodyMap) {
			h.Set("type", "Wap")
		})
	})

	rsp, err := s.client.V3TransactionH5(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("wechat: create H5 order: %w", err)
	}
	if rsp.Code != wechatV3.Success {
		return nil, fmt.Errorf("wechat: create H5 order failed: %s", rsp.Error)
	}
	if rsp.Response == nil || rsp.Response.H5Url == "" {
		return nil, fmt.Errorf("wechat: empty h5_url in response")
	}

	return &H5OrderResult{H5URL: rsp.Response.H5Url}, nil
}

// QueryOrderStatus queries the payment status of an existing order.
// Returns true when trade_state == "SUCCESS".
func (s *Service) QueryOrderStatus(ctx context.Context, outTradeNo string) (bool, error) {
	rsp, err := s.client.V3TransactionQueryOrder(ctx, wechatV3.OutTradeNo, outTradeNo)
	if err != nil {
		return false, fmt.Errorf("wechat: query order: %w", err)
	}
	if rsp.Code != wechatV3.Success {
		return false, fmt.Errorf("wechat: query order error: %s", rsp.Error)
	}
	if rsp.Response == nil {
		return false, nil
	}
	return rsp.Response.TradeState == "SUCCESS", nil
}

// ParseAndVerifyNotify parses a WeChat Pay V3 async payment notification from
// the HTTP request, verifies the signature with downloaded platform certs, and
// decrypts the ciphertext to a V3DecryptPayResult.
func (s *Service) ParseAndVerifyNotify(r *http.Request) (*wechatV3.V3DecryptPayResult, error) {
	notifyReq, err := wechatV3.V3ParseNotify(r)
	if err != nil {
		return nil, fmt.Errorf("wechat: parse notify: %w", err)
	}

	// Signature verification uses the cached platform public keys.
	if err = notifyReq.VerifySignByPKMap(s.client.WxPublicKeyMap()); err != nil {
		return nil, fmt.Errorf("wechat: verify notify signature: %w", err)
	}

	var result wechatV3.V3DecryptPayResult
	if err = notifyReq.DecryptCipherTextToStruct(s.cfg.APIv3Key, &result); err != nil {
		return nil, fmt.Errorf("wechat: decrypt notify: %w", err)
	}
	return &result, nil
}
