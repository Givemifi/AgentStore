// Package alipay wraps the Alipay web/WAP payment API.
// Automatically selects PC (trade.page.pay) or WAP (trade.wap.pay) based on
// the payer's User-Agent header.
package alipay

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"agentstore/internal/config"

	alipaySdk "github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay"
)

// Service wraps the gopay Alipay client.
type Service struct {
	client *alipaySdk.Client
	cfg    config.AlipayConfig
}

// New initialises the Alipay client.
// Returns nil, nil when the configuration is empty — treated as optional.
func New(cfg config.AlipayConfig) (*Service, error) {
	if cfg.AppID == "" || cfg.PrivateKey == "" || cfg.PublicKey == "" {
		return nil, nil
	}

	isProd := !cfg.IsSandbox
	client, err := alipaySdk.NewClient(cfg.AppID, cfg.PrivateKey, isProd)
	if err != nil {
		return nil, fmt.Errorf("alipay: init client: %w", err)
	}

	if cfg.NotifyURL != "" {
		client.SetNotifyUrl(cfg.NotifyURL)
	}
	if cfg.ReturnURL != "" {
		client.SetReturnUrl(cfg.ReturnURL)
	}

	return &Service{client: client, cfg: cfg}, nil
}

// PayOrderResult is returned by CreatePayOrder.
type PayOrderResult struct {
	PayURL   string // redirect URL for the payer
	IsMobile bool   // whether WAP (mobile) flow was used
}

// CreatePayOrder creates a payment redirect URL. The payment type (PC page vs
// WAP) is chosen automatically based on the payer's User-Agent.
//   - outTradeNo: unique merchant order ID (≤64 chars)
//   - subject: product name shown in the Alipay checkout
//   - totalYuan: amount in CNY yuan, e.g. "12.50"
//   - ua: payer's User-Agent header value
func (s *Service) CreatePayOrder(ctx context.Context, outTradeNo, subject, totalYuan, ua string) (*PayOrderResult, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", outTradeNo)
	bm.Set("total_amount", totalYuan)
	bm.Set("subject", subject)

	isMobile := isMobileUA(ua)

	var (
		payURL string
		err    error
	)
	if isMobile {
		payURL, err = s.client.TradeWapPay(ctx, bm)
	} else {
		payURL, err = s.client.TradePagePay(ctx, bm)
	}
	if err != nil {
		return nil, fmt.Errorf("alipay: create pay order: %w", err)
	}

	return &PayOrderResult{PayURL: payURL, IsMobile: isMobile}, nil
}

// QueryOrderStatus queries the payment status by outTradeNo.
// Returns true when trade_status is TRADE_SUCCESS or TRADE_FINISHED.
func (s *Service) QueryOrderStatus(ctx context.Context, outTradeNo string) (bool, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", outTradeNo)

	rsp, err := s.client.TradeQuery(ctx, bm)
	if err != nil {
		return false, fmt.Errorf("alipay: query order: %w", err)
	}
	if rsp.Response == nil {
		return false, nil
	}
	status := rsp.Response.TradeStatus
	return status == "TRADE_SUCCESS" || status == "TRADE_FINISHED", nil
}

// ParseAndVerifyNotify parses an Alipay async payment notification from the
// HTTP request, verifies the RSA2 signature using the configured public key,
// and returns the raw field map.
//
// Callers should check:
//
//	bm.GetString("trade_status") == "TRADE_SUCCESS" || "TRADE_FINISHED"
//	bm.GetString("out_trade_no")
func (s *Service) ParseAndVerifyNotify(r *http.Request) (gopay.BodyMap, error) {
	bm, err := alipaySdk.ParseNotifyToBodyMap(r)
	if err != nil {
		return nil, fmt.Errorf("alipay: parse notify: %w", err)
	}

	// VerifySign handles PEM wrapping of the raw public key internally.
	ok, err := alipaySdk.VerifySign(s.cfg.PublicKey, bm)
	if err != nil {
		return nil, fmt.Errorf("alipay: verify notify signature: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("alipay: notify signature invalid")
	}

	return bm, nil
}

// isMobileUA returns true when the User-Agent string indicates a mobile device.
func isMobileUA(ua string) bool {
	ua = strings.ToLower(ua)
	for _, kw := range []string{"mobile", "android", "iphone", "ipod", "windows phone"} {
		if strings.Contains(ua, kw) {
			return true
		}
	}
	return false
}
