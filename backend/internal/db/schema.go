package db

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// CollectionSchema pairs a collection name with its JSON Schema validator.
type CollectionSchema struct {
	Collection string
	Schema     bson.M
}

// AllSchemas returns the JSON Schema validators for all validated collections.
func AllSchemas() []CollectionSchema {
	return []CollectionSchema{
		usersSchema(),
		tenantsSchema(),
		tenantMembershipsSchema(),
		invitationsSchema(),
		plansSchema(),
		creditBundlesSchema(),
		financialTransactionsSchema(),
		paymentOrdersSchema(),
		webhooksSchema(),
		apiKeysSchema(),
		configVarsSchema(),
		announcementsSchema(),
		customPagesSchema(),
		messagesSchema(),
		usageEventsSchema(),
		ssoConnectionsSchema(),
		eventDefinitionsSchema(),
		conversationsSchema(),
		chatMessagesSchema(),
		llmConfigsSchema(),
		agentsSchema(),
		modelProvidersSchema(),
		modelConfigsSchema(),
		paymentConfigsSchema(),
		shareLinksSchema(),
		knowledgeDocumentsSchema(),
		knowledgeChunksSchema(),
		messageFeedbackSchema(),
		annotationsSchema(),
	}
}

// EnsureSchemaValidation applies JSON Schema validators to all validated
// collections using collMod with moderate validation level.
func (m *MongoDB) EnsureSchemaValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, cs := range AllSchemas() {
		// Ensure the collection exists (ignore "already exists" errors).
		_ = m.Database.CreateCollection(ctx, cs.Collection)

		cmd := bson.D{
			{Key: "collMod", Value: cs.Collection},
			{Key: "validator", Value: cs.Schema},
			{Key: "validationLevel", Value: "moderate"},
			{Key: "validationAction", Value: "error"},
		}

		if err := m.Database.RunCommand(ctx, cmd).Err(); err != nil {
			slog.Warn("failed to apply schema validation", "collection", cs.Collection, "error", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Individual collection schemas
// ---------------------------------------------------------------------------

func usersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "users",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"email", "displayName", "authMethods", "createdAt", "updatedAt"},
				"properties": bson.M{
					"email": bson.M{
						"bsonType": "string",
					},
					"displayName": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"authMethods": bson.M{
						"bsonType": "array",
						"minItems": 1,
						"items": bson.M{
							"bsonType": "string",
							"enum":     bson.A{"password", "google", "github", "microsoft", "magic_link", "passkey"},
						},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"emailVerified": bson.M{
						"bsonType": "bool",
					},
					"isActive": bson.M{
						"bsonType": "bool",
					},
					"themePreference": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"light", "dark", "system", ""},
					},
					"referralCode": bson.M{
						"bsonType":  "string",
						"maxLength": 16,
					},
					"referredBy": bson.M{
						"bsonType":  "string",
						"maxLength": 16,
					},
				},
			},
		},
	}
}

func tenantsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "tenants",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "slug", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"slug": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"isRoot": bson.M{
						"bsonType": "bool",
					},
					"isActive": bson.M{
						"bsonType": "bool",
					},
					"billingStatus": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"none", "active", "past_due", "canceled", ""},
					},
					"subscriptionCredits": bson.M{
						"bsonType": "long",
						"minimum":  0,
					},
					"purchasedCredits": bson.M{
						"bsonType": "long",
						"minimum":  0,
					},
					"bonusGrantedPlanIds": bson.M{
						"bsonType": "array",
						"items": bson.M{
							"bsonType": "objectId",
						},
					},
					"seatQuantity": bson.M{
						"bsonType": "int",
					},
					"defaultTextModelConfigId": bson.M{
						"bsonType": "objectId",
					},
					"defaultImageModelConfigId": bson.M{
						"bsonType": "objectId",
					},
					"defaultVideoModelConfigId": bson.M{
						"bsonType": "objectId",
					},
					"defaultEmbeddingModelConfigId": bson.M{
						"bsonType": "objectId",
					},
				},
			},
		},
	}
}

func tenantMembershipsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "tenant_memberships",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"userId", "tenantId", "role", "joinedAt", "updatedAt"},
				"properties": bson.M{
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"role": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"owner", "admin", "user"},
					},
					"joinedAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func invitationsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "invitations",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "email", "role", "token", "status", "invitedBy", "expiresAt", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"email": bson.M{
						"bsonType": "string",
					},
					"role": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"owner", "admin", "user"},
					},
					"token": bson.M{
						"bsonType": "string",
					},
					"status": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"pending", "accepted"},
					},
					"invitedBy": bson.M{
						"bsonType": "objectId",
					},
					"expiresAt": bson.M{
						"bsonType": "date",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func plansSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "plans",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "pricingModel", "creditResetPolicy", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"pricingModel": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"flat", "per_seat"},
					},
					"creditResetPolicy": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"reset", "accrue"},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"monthlyPriceCents": bson.M{
						"bsonType": "long",
						"minimum":  0,
					},
					"annualDiscountPct": bson.M{
						"bsonType": "int",
						"minimum":  0,
						"maximum":  100,
					},
					"trialDays": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
				},
			},
		},
	}
}

func creditBundlesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "credit_bundles",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "credits", "priceCents", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"credits": bson.M{
						"bsonType": "long",
						"minimum":  1,
					},
					"priceCents": bson.M{
						"bsonType": "long",
						"minimum":  1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func financialTransactionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "financial_transactions",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "type", "currency", "invoiceNumber", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"type": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"subscription", "credit_purchase", "refund"},
					},
					"currency": bson.M{
						"bsonType": "string",
					},
					"invoiceNumber": bson.M{
						"bsonType": "string",
					},
					"paymentProvider": bson.M{
						"bsonType": "string",
					},
					"outTradeNo": bson.M{
						"bsonType": "string",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

// paymentOrdersSchema validates pending domestic payment orders (WeChat/Alipay).
func paymentOrdersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "payment_orders",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"outTradeNo", "tenantId", "userId", "bundleId", "credits", "amountCents", "currency", "provider", "status", "createdAt"},
				"properties": bson.M{
					"outTradeNo": bson.M{
						"bsonType": "string",
					},
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"bundleId": bson.M{
						"bsonType": "objectId",
					},
					"credits": bson.M{
						"bsonType": "long",
					},
					"amountCents": bson.M{
						"bsonType": "long",
					},
					"currency": bson.M{
						"bsonType": "string",
					},
					"provider": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"wechat_h5", "alipay"},
					},
					"status": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"pending", "completed", "failed"},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func webhooksSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "webhooks",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "url", "secret", "secretPreview", "events", "createdBy", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"url": bson.M{
						"bsonType": "string",
					},
					"secret": bson.M{
						"bsonType": "string",
					},
					"secretPreview": bson.M{
						"bsonType": "string",
					},
					"events": bson.M{
						"bsonType": "array",
						"minItems": 1,
					},
					"createdBy": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func apiKeysSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "api_keys",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "keyHash", "keyPreview", "authority", "createdBy", "createdAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"keyHash": bson.M{
						"bsonType": "string",
					},
					"keyPreview": bson.M{
						"bsonType": "string",
					},
					"authority": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"admin", "user"},
					},
					"createdBy": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func configVarsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "config_vars",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "type", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"type": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"string", "numeric", "enum", "template"},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func announcementsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "announcements",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"title", "body", "createdAt", "updatedAt"},
				"properties": bson.M{
					"title": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"body": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func customPagesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "custom_pages",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"slug", "title", "createdAt", "updatedAt"},
				"properties": bson.M{
					"slug": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"title": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func messagesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "messages",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"userId", "subject", "body", "createdAt"},
				"properties": bson.M{
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"subject": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"body": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func usageEventsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "usage_events",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "type", "quantity", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"type": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"quantity": bson.M{
						"bsonType": "int",
						"minimum":  1,
					},
					"metadata": bson.M{
						"bsonType": "object",
						"additionalProperties": bson.M{
							"bsonType": "string",
						},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func ssoConnectionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "sso_connections",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "idpEntityId", "idpSsoUrl", "idpCertificate", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"idpEntityId": bson.M{
						"bsonType": "string",
					},
					"idpSsoUrl": bson.M{
						"bsonType": "string",
					},
					"idpCertificate": bson.M{
						"bsonType": "string",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func eventDefinitionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "event_definitions",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 128,
					},
					"description": bson.M{
						"bsonType":  "string",
						"maxLength": 256,
					},
					"parentId": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func conversationsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "conversations",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "agentId", "title", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"agentId": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"title": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func chatMessagesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "chat_messages",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "conversationId", "agentId", "role", "content", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"conversationId": bson.M{
						"bsonType": "objectId",
					},
					"agentId": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"role": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"user", "assistant"},
					},
					"content": bson.M{
						"bsonType": "string",
					},
					"creditsCharged": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
					"model": bson.M{
						"bsonType":  "string",
						"maxLength": 100,
					},
					"status": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"generating", "completed", "error", "interrupted", ""},
					},
					"attachmentCount": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
					"promptTokens": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
					"completionTokens": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func llmConfigsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "llm_configs",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"key", "apiKey", "baseURL", "model", "isActive", "createdAt", "updatedAt"},
				"properties": bson.M{
					"key": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"apiKey": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"baseURL": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"model": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"isActive": bson.M{
						"bsonType": "bool",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func agentsSchema() CollectionSchema {
	return CollectionSchema{Collection: "agents", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "name", "slug", "category", "description", "status", "visibility", "systemPrompt", "capabilities", "creditCost", "createdBy", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId":         bson.M{"bsonType": "objectId"},
			"name":             bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"slug":             bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"category":         bson.M{"bsonType": "string", "minLength": 1, "maxLength": 80},
			"description":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 500},
			"avatar":           bson.M{"bsonType": "string", "maxLength": 500},
			"icon":             bson.M{"bsonType": "string", "maxLength": 80},
			"color":            bson.M{"bsonType": "string", "maxLength": 32},
			"status":           bson.M{"bsonType": "string", "enum": bson.A{"draft", "published", "archived"}},
			"visibility":       bson.M{"bsonType": "string", "enum": bson.A{"private", "public"}},
			"systemPrompt":     bson.M{"bsonType": "string", "minLength": 1},
			"welcomeMessage":   bson.M{"bsonType": "string", "maxLength": 1000},
			"suggestedPrompts": bson.M{"bsonType": "array", "items": bson.M{"bsonType": "string", "maxLength": 300}},
			"capabilities":     bson.M{"bsonType": "array", "minItems": 1, "items": bson.M{"bsonType": "string", "enum": bson.A{"text_chat", "image_generation", "video_generation"}}},
			"creditCost": bson.M{"bsonType": "object", "properties": bson.M{
				"textMessageCredits":     bson.M{"bsonType": "int", "minimum": 0},
				"imageGenerationCredits": bson.M{"bsonType": "int", "minimum": 0},
				"videoGenerationCredits": bson.M{"bsonType": "int", "minimum": 0},
			}},
			"modelConfig": bson.M{"bsonType": "object", "properties": bson.M{
				"textModelId":  bson.M{"bsonType": "objectId"},
				"imageModelId": bson.M{"bsonType": "objectId"},
				"videoModelId": bson.M{"bsonType": "objectId"},
			}},
			"createdBy": bson.M{"bsonType": "objectId"},
			"createdAt": bson.M{"bsonType": "date"},
			"updatedAt": bson.M{"bsonType": "date"},
		},
	}}}
}

func modelProvidersSchema() CollectionSchema {
	return CollectionSchema{Collection: "model_providers", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "name", "providerType", "baseUrl", "apiKey", "enabled", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId":     bson.M{"bsonType": "objectId"},
			"name":         bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"providerType": bson.M{"bsonType": "string", "enum": bson.A{"openai_compatible", "anthropic", "gemini"}},
			"baseUrl":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 500},
			"apiKey":       bson.M{"bsonType": "string", "minLength": 1},
			"enabled":      bson.M{"bsonType": "bool"},
			"createdAt":    bson.M{"bsonType": "date"},
			"updatedAt":    bson.M{"bsonType": "date"},
		},
	}}}
}

func modelConfigsSchema() CollectionSchema {
	return CollectionSchema{Collection: "model_configs", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"tenantId", "providerId", "name", "displayName", "modality", "modelId", "enabled", "createdAt", "updatedAt"},
		"properties": bson.M{
			"tenantId":      bson.M{"bsonType": "objectId"},
			"providerId":    bson.M{"bsonType": "objectId"},
			"name":          bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
			"displayName":   bson.M{"bsonType": "string", "minLength": 1, "maxLength": 160},
			"modality":      bson.M{"bsonType": "string", "enum": bson.A{"text", "image", "video", "embedding"}},
			"modelId":       bson.M{"bsonType": "string", "minLength": 1, "maxLength": 200},
			"defaultParams": bson.M{"bsonType": "object"},
			"enabled":       bson.M{"bsonType": "bool"},
			"createdAt":     bson.M{"bsonType": "date"},
			"updatedAt":     bson.M{"bsonType": "date"},
		},
	}}}
}

func paymentConfigsSchema() CollectionSchema {
	return CollectionSchema{Collection: "payment_configs", Schema: bson.M{"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": bson.A{"key"},
		"properties": bson.M{
			"key":          bson.M{"bsonType": "string", "enum": bson.A{"wechat_pay", "alipay"}},
			"appId":        bson.M{"bsonType": "string"},
			"mchId":        bson.M{"bsonType": "string"},
			"apiV3Key":     bson.M{"bsonType": "string"},
			"privateKey":   bson.M{"bsonType": "string"},
			"certSerialNo": bson.M{"bsonType": "string"},
			"publicKey":    bson.M{"bsonType": "string"},
			"notifyUrl":    bson.M{"bsonType": "string"},
			"returnUrl":    bson.M{"bsonType": "string"},
			"isSandbox":    bson.M{"bsonType": "bool"},
			"enabled":      bson.M{"bsonType": "bool"},
			"createdAt":    bson.M{"bsonType": "date"},
			"updatedAt":    bson.M{"bsonType": "date"},
		},
	}}}
}

func shareLinksSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "share_links",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"token", "tenantId", "userId", "agentId", "agentName", "createdAt"},
				"properties": bson.M{
					"token": bson.M{
						"bsonType":  "string",
						"minLength": 16,
						"maxLength": 64,
					},
					"tenantId":  bson.M{"bsonType": "objectId"},
					"userId":    bson.M{"bsonType": "objectId"},
					"agentId":   bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"agentName": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 120},
					"title":     bson.M{"bsonType": "string", "maxLength": 200},
					"messages":  bson.M{"bsonType": "array"},
					"createdAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func knowledgeDocumentsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "knowledge_documents",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "agentId", "name", "sourceType", "status", "createdBy", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":   bson.M{"bsonType": "objectId"},
					"agentId":    bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"name":       bson.M{"bsonType": "string", "minLength": 1, "maxLength": 200},
					"sourceType": bson.M{"bsonType": "string", "enum": bson.A{"document", "text", "qa", "annotation"}},
					"status":     bson.M{"bsonType": "string", "enum": bson.A{"processing", "ready", "error"}},
					"errorMessage": bson.M{"bsonType": "string", "maxLength": 500},
					"chunkCount": bson.M{"bsonType": "int", "minimum": 0},
					"charCount":  bson.M{"bsonType": "int", "minimum": 0},
					"createdBy":  bson.M{"bsonType": "objectId"},
					"createdAt":  bson.M{"bsonType": "date"},
					"updatedAt":  bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func knowledgeChunksSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "knowledge_chunks",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "agentId", "documentId", "seq", "text", "createdAt"},
				"properties": bson.M{
					"tenantId":   bson.M{"bsonType": "objectId"},
					"agentId":    bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"documentId": bson.M{"bsonType": "objectId"},
					"seq":        bson.M{"bsonType": "int", "minimum": 0},
					"text":       bson.M{"bsonType": "string", "minLength": 1},
					"embedding":  bson.M{"bsonType": "array"},
					"createdAt":  bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func messageFeedbackSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "message_feedback",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "conversationId", "messageId", "agentId", "rating", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":       bson.M{"bsonType": "objectId"},
					"userId":         bson.M{"bsonType": "objectId"},
					"conversationId": bson.M{"bsonType": "objectId"},
					"messageId":      bson.M{"bsonType": "objectId"},
					"agentId":        bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"rating":         bson.M{"bsonType": "int", "enum": bson.A{1, -1}},
					"comment":        bson.M{"bsonType": "string", "maxLength": 2000},
					"createdAt":      bson.M{"bsonType": "date"},
					"updatedAt":      bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func annotationsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "annotations",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "conversationId", "messageId", "agentId", "annotatorId", "qualityScore", "status", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":       bson.M{"bsonType": "objectId"},
					"conversationId": bson.M{"bsonType": "objectId"},
					"messageId":      bson.M{"bsonType": "objectId"},
					"agentId":        bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"annotatorId":    bson.M{"bsonType": "objectId"},
					"qualityScore":   bson.M{"bsonType": "int", "minimum": 1, "maximum": 5},
					"issueTags":      bson.M{"bsonType": "array"},
					"idealAnswer":    bson.M{"bsonType": "string", "maxLength": 20000},
					"notes":          bson.M{"bsonType": "string", "maxLength": 2000},
					"status":         bson.M{"bsonType": "string", "enum": bson.A{"annotated", "promoted"}},
					"promotedDocumentId": bson.M{"bsonType": "objectId"},
					"createdAt":      bson.M{"bsonType": "date"},
					"updatedAt":      bson.M{"bsonType": "date"},
				},
			},
		},
	}
}
