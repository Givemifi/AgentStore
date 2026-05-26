package agents

import "testing"

func TestAgentCatalogUsesConcreteSuggestedPrompts(t *testing.T) {
	t.Parallel()

	expectedExamples := map[string][]string{
		"legal-expert": {
			"Review this contract clause for potential risks",
			"Explain the legal implications of this business decision",
			"Draft a simple NDA outline",
		},
		"tax-advisor": {
			"Help me understand tax implications for a cross-border SaaS business",
			"List deductible expenses for a small business",
			"Compare contractor vs employee tax considerations",
		},
		"marketing-copywriter": {
			"Write a landing page headline for my SaaS product",
			"Improve this ad copy",
			"Create 5 email subject lines for a product launch",
		},
		"customer-support": {
			"Draft a polite refund response",
			"Turn this angry customer message into a support ticket summary",
			"Create a help center article outline",
		},
		"telecom-business": {
			"Explain the difference between 5G CPE and mobile hotspot for business customers",
			"Draft a B2B proposal for enterprise connectivity",
			"Analyze risks in an IoT SIM card deployment",
		},
		"cross-border-ecommerce": {
			"Create a product listing optimization checklist",
			"Draft a supplier outreach email",
			"Explain common logistics risks for cross-border e-commerce",
		},
	}

	for agentID, expected := range expectedExamples {
		agent, err := GetAgentByID(agentID)
		if err != nil {
			t.Fatalf("GetAgentByID(%q) returned error: %v", agentID, err)
		}
		if agent == nil {
			t.Fatalf("expected agent %q to exist", agentID)
		}
		if len(agent.Examples) != len(expected) {
			t.Fatalf("agent %q returned %d examples, want %d", agentID, len(agent.Examples), len(expected))
		}

		for index, example := range expected {
			if agent.Examples[index] != example {
				t.Fatalf("agent %q example %d = %q, want %q", agentID, index, agent.Examples[index], example)
			}
		}
	}
}

func TestGetAllAgentsReturnsDefensiveCopy(t *testing.T) {
	agents := GetAllAgents()
	if len(agents) == 0 {
		t.Fatal("expected agents to be seeded")
	}

	originalID := agents[0].ID
	originalExample := agents[0].Examples[0]
	agents[0].ID = "mutated"
	agents[0].Examples[0] = "mutated example"

	freshAgents := GetAllAgents()
	if freshAgents[0].ID != originalID {
		t.Fatalf("expected catalog ID to remain %q, got %q", originalID, freshAgents[0].ID)
	}
	if freshAgents[0].Examples[0] != originalExample {
		t.Fatalf("expected catalog example to remain %q, got %q", originalExample, freshAgents[0].Examples[0])
	}
}

func TestGetAgentByIDReturnsDefensiveCopy(t *testing.T) {
	agent, err := GetAgentByID("legal-expert")
	if err != nil {
		t.Fatalf("GetAgentByID returned error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected legal-expert to exist")
	}

	originalExample := agent.Examples[0]
	agent.Examples[0] = "mutated example"

	freshAgent, err := GetAgentByID("legal-expert")
	if err != nil {
		t.Fatalf("GetAgentByID returned error: %v", err)
	}
	if freshAgent == nil {
		t.Fatal("expected legal-expert to exist")
	}
	if freshAgent.Examples[0] != originalExample {
		t.Fatalf("expected catalog example to remain %q, got %q", originalExample, freshAgent.Examples[0])
	}
}
