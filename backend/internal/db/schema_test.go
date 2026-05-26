package db

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func findSchema(collection string) (CollectionSchema, bool) {
	for _, schema := range AllSchemas() {
		if schema.Collection == collection {
			return schema, true
		}
	}
	return CollectionSchema{}, false
}

func TestAllSchemasIncludesAgentStoreCollections(t *testing.T) {
	for _, collection := range []string{"agents", "model_providers", "model_configs"} {
		if _, ok := findSchema(collection); !ok {
			t.Fatalf("expected schema for %s", collection)
		}
	}
}

func TestAgentsSchemaRequiresTenantScopedFields(t *testing.T) {
	schema, ok := findSchema("agents")
	if !ok {
		t.Fatal("agents schema missing")
	}
	jsonSchema := schema.Schema["$jsonSchema"].(bson.M)
	required := jsonSchema["required"].(bson.A)
	want := map[string]bool{"tenantId": true, "name": true, "slug": true, "status": true, "visibility": true, "systemPrompt": true}
	for _, item := range required {
		delete(want, item.(string))
	}
	if len(want) != 0 {
		t.Fatalf("missing required agent fields: %#v", want)
	}
}

func TestTenantsSchemaCreditsUseNonNegativeLongs(t *testing.T) {
	schema, ok := findSchema("tenants")
	if !ok {
		t.Fatal("tenants schema missing")
	}

	jsonSchema := schema.Schema["$jsonSchema"].(bson.M)
	properties := jsonSchema["properties"].(bson.M)

	for _, field := range []string{"subscriptionCredits", "purchasedCredits"} {
		property, ok := properties[field].(bson.M)
		if !ok {
			t.Fatalf("expected %s to be defined in tenants schema", field)
		}
		if got := property["bsonType"]; got != "long" {
			t.Fatalf("expected %s bsonType long, got %#v", field, got)
		}
		if got := property["minimum"]; got != 0 {
			t.Fatalf("expected %s minimum 0, got %#v", field, got)
		}
	}
}

func TestChatMessagesSchemaAllowsEmptyGeneratingPlaceholder(t *testing.T) {
	schema, ok := findSchema("chat_messages")
	if !ok {
		t.Fatal("chat_messages schema missing")
	}

	jsonSchema := schema.Schema["$jsonSchema"].(bson.M)
	properties := jsonSchema["properties"].(bson.M)
	content := properties["content"].(bson.M)
	status := properties["status"].(bson.M)

	if got := content["bsonType"]; got != "string" {
		t.Fatalf("expected content bsonType string, got %#v", got)
	}
	if _, hasMinLength := content["minLength"]; hasMinLength {
		t.Fatalf("expected content to omit minLength so generating placeholders may start empty, got %#v", content["minLength"])
	}
	if got := status["enum"]; got == nil {
		t.Fatal("expected status enum in chat_messages schema")
	}
}
