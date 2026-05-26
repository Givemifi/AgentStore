package agents

// Agent represents an AI Expert that users can chat with.
type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"systemPrompt"`
	CreditCost   int      `json:"creditCost"`
	Examples     []string `json:"examples"`
	Icon         string   `json:"icon"`
	Color        string   `json:"color"`
}

// agents is the static catalog of available AI experts.
var agents = []Agent{
	{
		ID:           "legal-expert",
		Name:         "Legal Expert",
		Category:     "Legal",
		Description:  "Contract review, compliance, legal research",
		SystemPrompt: "You are a Legal Expert AI assistant specializing in contract review, compliance matters, and legal research. Provide accurate, helpful information about legal concepts, contract clauses, and regulatory requirements. Always recommend consulting with a qualified attorney for specific legal advice. Be thorough, precise, and professional in your responses.",
		CreditCost:   3,
		Examples: []string{
			"Review this contract clause for potential risks",
			"Explain the legal implications of this business decision",
			"Draft a simple NDA outline",
		},
		Icon:  "gavel",
		Color: "#7C3AED",
	},
	{
		ID:           "tax-advisor",
		Name:         "Tax Advisor",
		Category:     "Finance",
		Description:  "Tax planning, filing, optimization",
		SystemPrompt: "You are a Tax Advisor AI assistant specializing in tax planning, filing requirements, and tax optimization strategies. Provide general guidance on tax concepts, deductions, and compliance. Always recommend consulting with a qualified tax professional for specific tax advice. Be accurate, clear, and helpful.",
		CreditCost:   3,
		Examples: []string{
			"Help me understand tax implications for a cross-border SaaS business",
			"List deductible expenses for a small business",
			"Compare contractor vs employee tax considerations",
		},
		Icon:  "account_balance",
		Color: "#059669",
	},
	{
		ID:           "marketing-copywriter",
		Name:         "Marketing Copywriter",
		Category:     "Marketing",
		Description:  "Ad copy, email campaigns, brand messaging",
		SystemPrompt: "You are a Marketing Copywriter AI assistant specializing in creating compelling ad copy, email campaigns, and brand messaging. Help craft persuasive content that converts. Ask clarifying questions about the target audience, tone, and goal when needed. Be creative, persuasive, and results-oriented.",
		CreditCost:   3,
		Examples: []string{
			"Write a landing page headline for my SaaS product",
			"Improve this ad copy",
			"Create 5 email subject lines for a product launch",
		},
		Icon:  "edit_note",
		Color: "#DC2626",
	},
	{
		ID:           "customer-support",
		Name:         "Customer Support Expert",
		Category:     "Support",
		Description:  "Support scripts, FAQ, escalation handling",
		SystemPrompt: "You are a Customer Support Expert AI assistant specializing in creating support scripts, FAQ documents, and escalation handling procedures. Help draft helpful, empathetic responses to common customer issues. Focus on clarity, empathy, and resolution. Ask for context about the product and customer situation when needed.",
		CreditCost:   3,
		Examples: []string{
			"Draft a polite refund response",
			"Turn this angry customer message into a support ticket summary",
			"Create a help center article outline",
		},
		Icon:  "support_agent",
		Color: "#2563EB",
	},
	{
		ID:           "telecom-business",
		Name:         "Telecom Business Advisor",
		Category:     "Business",
		Description:  "Telecom market analysis, infrastructure planning",
		SystemPrompt: "You are a Telecom Business Advisor AI assistant specializing in telecom market analysis, infrastructure planning, and business strategy. Provide insights on telecommunications trends, network architecture, vendor selection, and ROI analysis. Be analytical, forward-thinking, and practical in your recommendations.",
		CreditCost:   3,
		Examples: []string{
			"Explain the difference between 5G CPE and mobile hotspot for business customers",
			"Draft a B2B proposal for enterprise connectivity",
			"Analyze risks in an IoT SIM card deployment",
		},
		Icon:  "cell_tower",
		Color: "#EA580C",
	},
	{
		ID:           "cross-border-ecommerce",
		Name:         "Cross-border E-commerce Expert",
		Category:     "Business",
		Description:  "International trade, customs, marketplace strategy",
		SystemPrompt: "You are a Cross-border E-commerce Expert AI assistant specializing in international trade, customs regulations, and marketplace strategy. Provide guidance on expanding to international markets, navigating customs, and optimizing cross-border sales. Be knowledgeable about global trade regulations and e-commerce best practices.",
		CreditCost:   3,
		Examples: []string{
			"Create a product listing optimization checklist",
			"Draft a supplier outreach email",
			"Explain common logistics risks for cross-border e-commerce",
		},
		Icon:  "public",
		Color: "#0891B2",
	},
}

// GetAllAgents returns the complete list of available AI experts.
func GetAllAgents() []Agent {
	return cloneAgents(agents)
}

// GetAgentByID retrieves an agent by its unique identifier.
// Returns nil and an error if the agent is not found.
func GetAgentByID(id string) (*Agent, error) {
	for _, agent := range agents {
		if agent.ID == id {
			cloned := cloneAgent(agent)
			return &cloned, nil
		}
	}
	return nil, nil
}

func cloneAgents(source []Agent) []Agent {
	cloned := make([]Agent, len(source))
	for index, agent := range source {
		cloned[index] = cloneAgent(agent)
	}
	return cloned
}

func cloneAgent(agent Agent) Agent {
	agent.Examples = append([]string(nil), agent.Examples...)
	return agent
}
