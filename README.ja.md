# AgentStore

[![CI](https://github.com/Givemifi/AgentStore/actions/workflows/ci.yml/badge.svg)](https://github.com/Givemifi/AgentStore/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Givemifi/AgentStore/branch/master/graph/badge.svg)](https://codecov.io/gh/Givemifi/AgentStore)
[![Go Report Card](https://goreportcard.com/badge/github.com/Givemifi/AgentStore)](https://goreportcard.com/report/github.com/Givemifi/AgentStore)

**言語 / Language:** [English](README.md) | [简体中文](README.zh-CN.md) | 日本語

**企業向けオープンソース AI エージェント マーケットプレイス。SaaS 基盤の上に構築。**

AgentStore は、キュレートされた AI エージェントのマーケットプレイスを素早く立ち上げるためのプラットフォームです。ユーザーはエージェントを探し、対話し、クレジット残高を確認し、クレジットを購入できます。プロダクション対応の SaaS 基盤として、マルチテナント、認証、RBAC、ホワイトラベルブランディング、Stripe 課金、API キー、Webhook、管理ダッシュボード、ヘルス監視、クレジットベースの使用量トラッキング、プロダクトアナリティクスを内包しています。

---

## なぜ AgentStore を作ったのか

すべての SaaS プロダクトは同じ退屈な基盤が必要です：ユーザーアカウント、チーム、ロール、認証、管理ダッシュボード、課金、使用量制限、ブランディング、Webhook、API キー。従来この基盤を構築するには、本来のプロダクトコードを 1 行も書く前に数週間かかっていました。

AgentStore はそれを排除します。フォークして、AI エージェントに任せて、以下をすべて処理済みの基盤の上でプロダクト開発を始められます：

- マルチテナント分離 + ロールベースアクセス制御
- JWT 認証（リフレッシュトークンローテーション付き）
- Google / GitHub / Microsoft OAuth 統合
- マジックリンク（パスワードレス）認証
- MFA/TOTP + リカバリコード
- メール認証とパスワードリセット
- チーム招待とメンバー管理
- Stripe 課金（サブスクリプション、席数課金、トライアル、クレジットバンドル）
- プラン権限と課金強制ミドルウェア
- ホワイトラベルブランディング（カスタムテーマ、ロゴ、ランディングページ）
- API キー認証（管理者 & ユーザーの 2 スコープ）
- 出力 Webhook（19 イベントタイプ、HMAC-SHA256 署名）
- クレジット使用量トラッキング
- プロモーションコードと Stripe クーポン管理
- プロダクトアナリティクスダッシュボード
- テレメトリイベントシステム
- フル管理画面
- 組み込み API ドキュメント
- リアルタイムシステムヘルス監視
- MCP（Model Context Protocol）サーバー
- CLI 管理ツール
- Fly.io への本番デプロイ対応

---

## 主な機能

### 認証
- パスワード + メール認証
- Google / GitHub / Microsoft OAuth
- マジックリンク（パスワードレス）
- MFA/TOTP（リカバリコード付き）
- Passkeys（WebAuthn）
- セッション管理（マルチデバイスログアウト）

### エージェント マーケットプレイス
- 23 のプリセット精選エージェント（法律・金融・マーケティング・カスタマーサポート・コーディング・ライティング・翻訳など 7 カテゴリ）
- SSE ストリーミングチャット（Markdown レンダリング、コードブロックコピー）
- マルチモーダル：音声（ホールドトゥトーク）、画像（Vision）、ドキュメント（PDF / Word / TXT）
- 公開エージェントカタログ（`/agents`）：ログイン不要で閲覧可能
- 会話共有リンク

### 課金 & クレジット
- サブスクリプションクレジット（月次付与）
- クレジットパック購入（一括）
- エージェントコストに基づく会話ごとのクレジット消費
- 残高不足リアルタイム遮断（402）
- Stripe Checkout / サブスクリプション / Customer Portal
- WeChat Pay H5 + Alipay ウェブ決済（クレジットパック）
- プロモーションコードと Stripe クーポン

### 管理ダッシュボード
- ユーザー / テナント / メンバー / 招待管理
- プランとクレジットパックの設定
- 課金と請求書
- ブランドカスタマイズ（アプリ名、カラーテーマ、ロゴ、カスタムページ）
- システムヘルスとメトリクス
- 構造化システムログ
- API キー管理
- 出力 Webhook（19 イベント、AES-256-GCM 暗号化ペイロード）
- プロダクトアナリティクスとテレメトリダッシュボード
- 本番稼働チェックリスト
- LLM 設定（ランタイムでプロバイダー変更可能）

---

## 技術スタック

| レイヤー | 技術 |
|---|---|
| バックエンド | Go 1.25, gorilla/mux |
| フロントエンド | React 19, TypeScript, Vite 7, Tailwind CSS 4 |
| データベース | MongoDB（Atlas またはローカル） |
| 認証 | JWT, bcrypt, OAuth, Magic Link, TOTP MFA |
| 課金 | Stripe (stripe-go v82) |
| メール | Resend |
| デプロイ | Docker, Fly.io |

---

## クイックスタート

### 🚀 ワンコマンドデプロイ（Docker Compose — 推奨）

**必要なもの：** [Docker Desktop](https://docs.docker.com/get-docker/)（Compose を含む）

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

cp .env.docker.example .env
# .env を開き、以下を設定:
#   JWT_ACCESS_SECRET    ← openssl rand -hex 32 の出力
#   JWT_REFRESH_SECRET   ← openssl rand -hex 32 の出力
#   WEBHOOK_ENCRYPTION_KEY ← openssl rand -hex 32 の出力
#   OPENAI_* (オプション — 空欄にすると後で管理画面から設定)

docker compose up -d
# → http://localhost:8080 でセットアップウィザードを開く
```

> **ウィザードをスキップする場合**は、`.env` に `AGENTSTORE_SETUP_ORG`、`AGENTSTORE_SETUP_NAME`、`AGENTSTORE_SETUP_EMAIL`、`AGENTSTORE_SETUP_PASSWORD` を設定してください。

### 開発モード（ソースから）

```bash
./scripts/setup.sh
# ターミナル 1: cd backend && go run ./cmd/server
# ターミナル 2: cd frontend && npm install && npm run dev
# 初回のみ: cd backend && go run ./cmd/agentstore setup
```

---

## 本番デプロイ

詳細は [docs/DEPLOYMENT.ja.md](docs/DEPLOYMENT.ja.md) を参照してください。

### Fly.io（推奨）

```bash
fly auth login && fly apps create agentstore
fly secrets set JWT_ACCESS_SECRET=$(openssl rand -hex 32) \
  JWT_REFRESH_SECRET=$(openssl rand -hex 32) \
  WEBHOOK_ENCRYPTION_KEY=$(openssl rand -hex 32) \
  MONGODB_URI="mongodb+srv://..." DATABASE_NAME=agentstore \
  FRONTEND_URL=https://your-app.fly.dev APP_NAME=AgentStore
fly deploy
```

### Docker（任意ホスト）

```bash
docker pull ghcr.io/givemifi/agentstore:latest
docker run -d -p 8080:8080 --env-file .env \
  -e MONGODB_URI="your-uri" ghcr.io/givemifi/agentstore:latest
```

---

## ライセンス

MIT License — Copyright (c) 2026 Givemifi

[LICENSE](LICENSE) を参照してください。
