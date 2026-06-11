# AgentStore デプロイガイド

AgentStore を本番環境にデプロイする方法を説明します。

**言語：** [English](DEPLOYMENT.md) | [简体中文](DEPLOYMENT.zh-CN.md) | 日本語

---

## 実行モデル

プロダクションコンテナは Go バックエンドバイナリとビルド済み React フロントエンドを 1 つのイメージに含み、`SERVER_PORT`（Fly.io 推奨: `8080`）でリッスンします。nginx やフロントエンド専用サーバーは不要です。

## 事前ビルド済みイメージ

リリースごとに GHCR へ自動的にイメージが発行されます：

```bash
docker pull ghcr.io/givemifi/agentstore:latest
docker pull ghcr.io/givemifi/agentstore:1.3.0
```

---

## Docker Compose（セルフホストに推奨）

Go や Node.js のインストール不要。MongoDB は自動で起動します。

### 1. クローンと設定

```bash
git clone https://github.com/Givemifi/AgentStore.git
cd AgentStore

cp .env.docker.example .env
```

`.env` を開き、必須フィールドを設定します：

```bash
# 必須シークレット（各コマンドで生成）
JWT_ACCESS_SECRET=$(openssl rand -hex 32)
JWT_REFRESH_SECRET=$(openssl rand -hex 32)
WEBHOOK_ENCRYPTION_KEY=$(openssl rand -hex 32)

# オプション：起動直後からチャットを有効にする
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com
OPENAI_MODEL=gpt-4o
```

### 2. 起動

```bash
docker compose up -d
```

事前ビルド済みイメージを使用する場合は、`docker-compose.yml` の `build: .` をコメントアウトし `image:` 行のコメントを外してください。

### 3. 初回セットアップ

ブラウザで `http://localhost:8080` を開き、セットアップウィザードで管理者アカウントを作成します。

**自動初期化（CI/CD 向け）：** 以下の変数を `.env` に設定すると、起動時にウィザードをスキップして自動でアカウントが作成されます：

```bash
AGENTSTORE_SETUP_ORG=My Company
AGENTSTORE_SETUP_NAME=Taro Yamada
AGENTSTORE_SETUP_EMAIL=admin@example.com
AGENTSTORE_SETUP_PASSWORD=YourSecurePass123!
```

### 4. LLM の設定（環境変数未設定の場合）

ログイン → Admin → LLM Configuration → プロバイダー URL、API キー、モデル名を入力。

---

## Fly.io デプロイ

### 1. Fly CLI のインストール

```bash
curl -L https://fly.io/install.sh | sh
flyctl auth login
```

### 2. アプリ作成

```bash
flyctl apps create your-app-name --org your-org
```

### 3. 必須シークレットの設定

```bash
flyctl secrets set \
  SERVER_PORT="8080" \
  DATABASE_NAME="agentstore" \
  MONGODB_URI="mongodb+srv://..." \
  JWT_ACCESS_SECRET="$(openssl rand -hex 32)" \
  JWT_REFRESH_SECRET="$(openssl rand -hex 32)" \
  WEBHOOK_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  FRONTEND_URL="https://your-app-name.fly.dev" \
  APP_NAME="AgentStore" \
  FROM_EMAIL="noreply@yourdomain.com" \
  FROM_NAME="AgentStore"
```

### 4. オプション統合のシークレット設定

```bash
flyctl secrets set \
  OPENAI_API_KEY="sk-..." \
  OPENAI_BASE_URL="https://api.openai.com" \
  OPENAI_MODEL="gpt-4o" \
  RESEND_API_KEY="re_..." \
  STRIPE_SECRET_KEY="sk_live_..." \
  STRIPE_PUBLISHABLE_KEY="pk_live_..." \
  STRIPE_WEBHOOK_SECRET="whsec_..."
```

### 5. デプロイ

```bash
flyctl deploy
```

### 6. ヘルスチェック確認

```bash
curl -fsS https://your-app-name.fly.dev/health
```

HTTP 200 が返れば成功です。

---

## 初回初期化

デプロイ後にブラウザでアプリ URL を開くと、未初期化の場合はセットアップウィザードが表示されます。

**CLI による初期化（MongoDB にローカルからアクセスできる場合）：**

```bash
set -a && source .env.production && set +a
export AGENTSTORE_ENV=prod
cd backend
go run ./cmd/agentstore setup
```

---

## 必要な環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| `SERVER_PORT` | はい | Fly.io では `8080` |
| `DATABASE_NAME` | はい | MongoDB データベース名 |
| `MONGODB_URI` | はい | MongoDB 接続文字列 |
| `JWT_ACCESS_SECRET` | はい | `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | はい | `openssl rand -hex 32` |
| `WEBHOOK_ENCRYPTION_KEY` | はい | `openssl rand -hex 32` |
| `FRONTEND_URL` | はい | 公開 HTTPS URL |
| `APP_NAME` | はい | アプリ表示名 |
| `OPENAI_API_KEY` | オプション | LLM プロバイダーキー |
| `OPENAI_BASE_URL` | オプション | LLM プロバイダーの URL |
| `OPENAI_MODEL` | オプション | モデル名（例：`gpt-4o`） |

3 つの `OPENAI_*` をすべて設定すると初回起動時に LLM 設定が自動作成されます。空欄の場合は管理画面から手動設定してください。

その他のオプション変数（OAuth / Stripe / WeChat Pay / Alipay / DataDog）は `.env.docker.example` を参照してください。

---

## Stripe Webhook の設定

課金機能を使用する場合は Stripe Webhook エンドポイントを設定します：

```
https://your-app.com/api/billing/webhook
```

以下のイベントをサブスクライブしてください：
- `checkout.session.completed`
- `customer.subscription.created` / `updated` / `deleted`
- `invoice.payment_succeeded` / `invoice.payment_failed`
- `charge.refunded` / `charge.dispute.created`

Signing Secret を `STRIPE_WEBHOOK_SECRET` に設定します。
