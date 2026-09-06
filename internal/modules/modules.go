package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/v-pat/fiberforge/internal/engine"
	"github.com/v-pat/fiberforge/internal/schema"
)

// Module represents a pre-packaged feature recipe.
type Module struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Models      []schema.Model `json:"models"`
}

var catalog = map[string]*Module{
	"stripe-billing": {
		Name:        "stripe-billing",
		Title:       "Stripe Billing & Subscriptions",
		Description: "Stripe customer, subscription, plan, and invoice management with webhook handling",
		Category:    "Payments",
		Models: []schema.Model{
			{
				Name:          "StripeCustomer",
				Endpoint:      "stripe-customers",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "CustomerID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Email", Type: schema.TypeString, Required: true},
					{Name: "Currency", Type: schema.TypeString, Default: strPtr("usd")},
				},
			},
			{
				Name:          "Subscription",
				Endpoint:      "subscriptions",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "SubscriptionID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "CustomerID", Type: schema.TypeString, Required: true, Index: true},
					{Name: "PlanID", Type: schema.TypeString, Required: true},
					{Name: "Status", Type: schema.TypeEnum, Values: []string{"active", "past_due", "canceled", "incomplete"}, Required: true},
					{Name: "CurrentPeriodEnd", Type: schema.TypeTime},
				},
				Relationships: []schema.Relationship{
					{Type: schema.BelongsTo, Model: "StripeCustomer"},
				},
			},
			{
				Name:          "Invoice",
				Endpoint:      "invoices",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "InvoiceID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "AmountPaid", Type: schema.TypeInt64, Required: true},
					{Name: "Currency", Type: schema.TypeString, Required: true},
					{Name: "Status", Type: schema.TypeEnum, Values: []string{"paid", "open", "uncollectible", "void"}, Required: true},
					{Name: "HostedInvoiceURL", Type: schema.TypeString},
				},
				Relationships: []schema.Relationship{
					{Type: schema.BelongsTo, Model: "StripeCustomer"},
				},
			},
		},
	},
	"ai-inference": {
		Name:        "ai-inference",
		Title:       "AI Inference & SSE Token Streaming",
		Description: "OpenAI-compatible chat completions API with SSE token streaming, model registry, and token usage logs",
		Category:    "AI & LLM",
		Models: []schema.Model{
			{
				Name:     "AIModel",
				Endpoint: "ai-models",
				Fields: []schema.Field{
					{Name: "ModelID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Provider", Type: schema.TypeEnum, Values: []string{"openai", "anthropic", "ollama", "local"}, Required: true},
					{Name: "MaxTokens", Type: schema.TypeInt, Default: strPtr("4096")},
					{Name: "Active", Type: schema.TypeBool, Default: strPtr("true")},
				},
			},
			{
				Name:          "PromptTemplate",
				Endpoint:      "prompt-templates",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString, Required: true},
					{Name: "SystemPrompt", Type: schema.TypeText, Required: true},
					{Name: "Temperature", Type: schema.TypeFloat, Default: strPtr("0.7")},
				},
			},
			{
				Name:          "TokenUsageLog",
				Endpoint:      "token-usage-logs",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "PromptTokens", Type: schema.TypeInt, Required: true},
					{Name: "CompletionTokens", Type: schema.TypeInt, Required: true},
					{Name: "TotalTokens", Type: schema.TypeInt, Required: true},
					{Name: "ModelName", Type: schema.TypeString, Required: true},
				},
			},
		},
	},
	"vector-search": {
		Name:        "vector-search",
		Title:       "Vector Similarity Search",
		Description: "Embeddings, vector document indexing, and similarity search queries",
		Category:    "AI & LLM",
		Models: []schema.Model{
			{
				Name:     "EmbeddingDocument",
				Endpoint: "embedding-documents",
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString, Required: true},
					{Name: "Content", Type: schema.TypeText, Required: true},
					{Name: "VectorID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Dimensions", Type: schema.TypeInt, Default: strPtr("1536")},
				},
			},
		},
	},
	"rbac": {
		Name:        "rbac",
		Title:       "Role-Based Access Control (RBAC)",
		Description: "Dynamic roles, permissions, and role assignment middleware",
		Category:    "Security",
		Models: []schema.Model{
			{
				Name:     "Role",
				Endpoint: "roles",
				Fields: []schema.Field{
					{Name: "Name", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Description", Type: schema.TypeString},
				},
			},
			{
				Name:     "Permission",
				Endpoint: "permissions",
				Fields: []schema.Field{
					{Name: "Code", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Resource", Type: schema.TypeString, Required: true},
					{Name: "Action", Type: schema.TypeString, Required: true},
				},
			},
		},
	},
	"auth-oauth2": {
		Name:        "auth-oauth2",
		Title:       "OAuth2 Social Authentication",
		Description: "Google, GitHub, and Apple OAuth2 login handlers and user account linkings",
		Category:    "Security",
		Models: []schema.Model{
			{
				Name:          "OAuthAccount",
				Endpoint:      "oauth-accounts",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Provider", Type: schema.TypeEnum, Values: []string{"google", "github", "apple"}, Required: true},
					{Name: "ProviderUserID", Type: schema.TypeString, Required: true, Index: true},
					{Name: "AccessToken", Type: schema.TypeString, Sensitive: true},
				},
			},
		},
	},
	"otp-2fa": {
		Name:        "otp-2fa",
		Title:       "TOTP Two-Factor Authentication",
		Description: "Google Authenticator 2FA TOTP setup, QR codes, and recovery codes",
		Category:    "Security",
		Models: []schema.Model{
			{
				Name:          "User2FA",
				Endpoint:      "user-2fa",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Secret", Type: schema.TypeString, Required: true, Sensitive: true},
					{Name: "Enabled", Type: schema.TypeBool, Default: strPtr("false")},
					{Name: "RecoveryCodes", Type: schema.TypeJSON},
				},
			},
		},
	},
	"api-keys": {
		Name:        "api-keys",
		Title:       "Developer API Keys",
		Description: "Hashed API key generation, prefixing, scopes, and middleware authentication",
		Category:    "Security",
		Models: []schema.Model{
			{
				Name:          "APIKey",
				Endpoint:      "api-keys",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "KeyPrefix", Type: schema.TypeString, Required: true, Index: true},
					{Name: "KeyHash", Type: schema.TypeString, Required: true, Sensitive: true},
					{Name: "Name", Type: schema.TypeString, Required: true},
					{Name: "Scopes", Type: schema.TypeJSON},
					{Name: "ExpiresAt", Type: schema.TypeTime},
				},
			},
		},
	},
	"s3-storage": {
		Name:        "s3-storage",
		Title:       "AWS S3 & Cloud Storage Uploads",
		Description: "File attachment models, presigned URLs, and S3 multipart upload management",
		Category:    "Storage",
		Models: []schema.Model{
			{
				Name:          "FileAttachment",
				Endpoint:      "files",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "FileName", Type: schema.TypeString, Required: true},
					{Name: "ObjectKey", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "MimeType", Type: schema.TypeString, Required: true},
					{Name: "SizeBytes", Type: schema.TypeInt64, Required: true},
					{Name: "PublicURL", Type: schema.TypeString},
				},
			},
		},
	},
	"notifications": {
		Name:        "notifications",
		Title:       "Real-Time Notifications (WebSocket/SSE)",
		Description: "In-app notifications, user preferences, and real-time streaming",
		Category:    "Messaging",
		Models: []schema.Model{
			{
				Name:          "Notification",
				Endpoint:      "notifications",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Title", Type: schema.TypeString, Required: true},
					{Name: "Body", Type: schema.TypeText, Required: true},
					{Name: "Read", Type: schema.TypeBool, Default: strPtr("false")},
					{Name: "Type", Type: schema.TypeEnum, Values: []string{"info", "warning", "success", "alert"}, Default: strPtr("info")},
				},
			},
		},
	},
	"email-templates": {
		Name:        "email-templates",
		Title:       "Transactional Email Delivery",
		Description: "Email logging, HTML template rendering, and Resend/SendGrid/SMTP delivery",
		Category:    "Messaging",
		Models: []schema.Model{
			{
				Name:     "EmailLog",
				Endpoint: "email-logs",
				Fields: []schema.Field{
					{Name: "ToEmail", Type: schema.TypeString, Required: true, Index: true},
					{Name: "Subject", Type: schema.TypeString, Required: true},
					{Name: "Status", Type: schema.TypeEnum, Values: []string{"queued", "sent", "failed"}, Required: true},
					{Name: "ErrorMessage", Type: schema.TypeString},
				},
			},
		},
	},
	"webhooks-engine": {
		Name:        "webhooks-engine",
		Title:       "Outgoing Developer Webhooks",
		Description: "Webhook endpoints, HMAC signature signing, and delivery retry logging",
		Category:    "Messaging",
		Models: []schema.Model{
			{
				Name:          "WebhookEndpoint",
				Endpoint:      "webhook-endpoints",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "TargetURL", Type: schema.TypeString, Required: true},
					{Name: "Secret", Type: schema.TypeString, Required: true, Sensitive: true},
					{Name: "Events", Type: schema.TypeJSON},
					{Name: "Active", Type: schema.TypeBool, Default: strPtr("true")},
				},
			},
		},
	},
	"audit-log": {
		Name:        "audit-log",
		Title:       "Compliance Audit Trail",
		Description: "Immutable audit log tracking system actions, user mutations, IP addresses, and payload diffs",
		Category:    "Governance",
		Models: []schema.Model{
			{
				Name:     "AuditLog",
				Endpoint: "audit-logs",
				Fields: []schema.Field{
					{Name: "ActorID", Type: schema.TypeString, Required: true, Index: true},
					{Name: "Action", Type: schema.TypeString, Required: true},
					{Name: "ResourceType", Type: schema.TypeString, Required: true},
					{Name: "ResourceID", Type: schema.TypeString, Required: true},
					{Name: "IPAddress", Type: schema.TypeString},
					{Name: "UserAgent", Type: schema.TypeString},
				},
			},
		},
	},
	"multi-tenant": {
		Name:        "multi-tenant",
		Title:       "SaaS Multi-Tenancy Isolation",
		Description: "Organizations, tenant memberships, and automated data scoping",
		Category:    "Governance",
		Models: []schema.Model{
			{
				Name:     "Organization",
				Endpoint: "organizations",
				Fields: []schema.Field{
					{Name: "Name", Type: schema.TypeString, Required: true},
					{Name: "Slug", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Domain", Type: schema.TypeString, Unique: true},
				},
			},
		},
	},
	"feature-flags": {
		Name:        "feature-flags",
		Title:       "Dynamic Feature Toggling",
		Description: "Feature flag rules, percentage rollouts, and runtime flag evaluation",
		Category:    "Governance",
		Models: []schema.Model{
			{
				Name:     "FeatureFlag",
				Endpoint: "feature-flags",
				Fields: []schema.Field{
					{Name: "Key", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Description", Type: schema.TypeString},
					{Name: "Enabled", Type: schema.TypeBool, Default: strPtr("false")},
					{Name: "RolloutPercentage", Type: schema.TypeInt, Default: strPtr("100")},
				},
			},
		},
	},
	"queues-asynq": {
		Name:        "queues-asynq",
		Title:       "Redis Background Task Queues",
		Description: "Async task queue state, delayed jobs, and worker execution logs",
		Category:    "Workers & Data",
		Models: []schema.Model{
			{
				Name:     "TaskStateLog",
				Endpoint: "task-logs",
				Fields: []schema.Field{
					{Name: "TaskID", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Queue", Type: schema.TypeString, Required: true},
					{Name: "State", Type: schema.TypeEnum, Values: []string{"pending", "active", "completed", "failed"}, Required: true},
					{Name: "Error", Type: schema.TypeText},
				},
			},
		},
	},
	"search-meilisearch": {
		Name:        "search-meilisearch",
		Title:       "Meilisearch / Full-Text Search Sync",
		Description: "Full-text search index synchronization and query configuration",
		Category:    "Workers & Data",
		Models: []schema.Model{
			{
				Name:     "SearchIndexConfig",
				Endpoint: "search-index-configs",
				Fields: []schema.Field{
					{Name: "IndexName", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "SearchableAttributes", Type: schema.TypeJSON},
					{Name: "FilterableAttributes", Type: schema.TypeJSON},
				},
			},
		},
	},
	"rate-limiter-redis": {
		Name:        "rate-limiter-redis",
		Title:       "Redis Sliding-Window Rate Limiting",
		Description: "Distributed rate limit policies and quota tiers",
		Category:    "Workers & Data",
		Models: []schema.Model{
			{
				Name:     "RateLimitPolicy",
				Endpoint: "rate-limit-policies",
				Fields: []schema.Field{
					{Name: "RoutePattern", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "MaxRequests", Type: schema.TypeInt, Required: true},
					{Name: "WindowSeconds", Type: schema.TypeInt, Required: true},
				},
			},
		},
	},
	"caching-redis": {
		Name:        "caching-redis",
		Title:       "HTTP Response & Query Cache",
		Description: "Cache key invalidation rules and response caching policies",
		Category:    "Workers & Data",
		Models: []schema.Model{
			{
				Name:     "CacheRule",
				Endpoint: "cache-rules",
				Fields: []schema.Field{
					{Name: "PathPrefix", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "TTLSeconds", Type: schema.TypeInt, Required: true},
					{Name: "Tags", Type: schema.TypeJSON},
				},
			},
		},
	},
	"cron-scheduler": {
		Name:        "cron-scheduler",
		Title:       "In-Process Cron Job Scheduler",
		Description: "Scheduled cron jobs, execution frequency, and task history",
		Category:    "Workers & Data",
		Models: []schema.Model{
			{
				Name:     "CronJob",
				Endpoint: "cron-jobs",
				Fields: []schema.Field{
					{Name: "Name", Type: schema.TypeString, Required: true, Unique: true},
					{Name: "Schedule", Type: schema.TypeString, Required: true}, // e.g. "0 0 * * *"
					{Name: "Enabled", Type: schema.TypeBool, Default: strPtr("true")},
					{Name: "LastRunAt", Type: schema.TypeTime},
				},
			},
		},
	},
	"comments-reactions": {
		Name:        "comments-reactions",
		Title:       "Threaded Comments & Emoji Reactions",
		Description: "Polymorphic threaded comments with replies and emoji reaction counts",
		Category:    "Social",
		Models: []schema.Model{
			{
				Name:          "Comment",
				Endpoint:      "comments",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Body", Type: schema.TypeText, Required: true},
					{Name: "ParentID", Type: schema.TypeString, Index: true},
				},
			},
			{
				Name:          "Reaction",
				Endpoint:      "reactions",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Emoji", Type: schema.TypeString, Required: true},
					{Name: "TargetType", Type: schema.TypeString, Required: true},
					{Name: "TargetID", Type: schema.TypeString, Required: true},
				},
			},
		},
	},
	"activity-feed": {
		Name:        "activity-feed",
		Title:       "User Activity Feed & Follows",
		Description: "User follower graphs and activity stream generation",
		Category:    "Social",
		Models: []schema.Model{
			{
				Name:          "UserFollow",
				Endpoint:      "user-follows",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "FollowerID", Type: schema.TypeString, Required: true, Index: true},
					{Name: "FollowingID", Type: schema.TypeString, Required: true, Index: true},
				},
			},
			{
				Name:          "ActivityItem",
				Endpoint:      "activity-items",
				AuthProtected: true,
				Fields: []schema.Field{
					{Name: "Verb", Type: schema.TypeString, Required: true},
					{Name: "ObjectSummary", Type: schema.TypeString, Required: true},
				},
			},
		},
	},
	"analytics-events": {
		Name:        "analytics-events",
		Title:       "Product Analytics & Event Ingestion",
		Description: "Event logging, user sessions, and analytics aggregation",
		Category:    "Social",
		Models: []schema.Model{
			{
				Name:     "AnalyticsEvent",
				Endpoint: "analytics-events",
				Fields: []schema.Field{
					{Name: "EventName", Type: schema.TypeString, Required: true, Index: true},
					{Name: "SessionID", Type: schema.TypeString, Index: true},
					{Name: "Properties", Type: schema.TypeJSON},
				},
			},
		},
	},
}

func strPtr(s string) *string {
	return &s
}

// List returns all available modules in the catalog.
func List() []*Module {
	var list []*Module
	for _, m := range catalog {
		list = append(list, m)
	}
	return list
}

// Get fetches a module by name.
func Get(name string) (*Module, error) {
	name = strings.ToLower(name)
	m, ok := catalog[name]
	if !ok {
		return nil, fmt.Errorf("unknown module %q", name)
	}
	return m, nil
}

// Apply appends all models in a module to an existing project target directory.
func Apply(targetDir string, moduleName string) ([]string, error) {
	added, _, err := ApplyWithOptions(targetDir, moduleName, false)
	return added, err
}

// ApplyWithOptions appends all models in a module with optional dry-run mode.
// Returns (addedModelNames, affectedFiles, error).
func ApplyWithOptions(targetDir string, moduleName string, dryRun bool) ([]string, []string, error) {
	mod, err := Get(moduleName)
	if err != nil {
		return nil, nil, err
	}

	var added []string
	var allFiles []string
	for _, model := range mod.Models {
		files, err := engine.AddModelWithOptions(targetDir, model, dryRun)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return nil, nil, fmt.Errorf("failed to apply model %s from module %s: %w", model.Name, moduleName, err)
		}
		added = append(added, model.Name)
		allFiles = append(allFiles, files...)
	}

	// Helper pkg path calculation
	helperPkgMap := map[string]string{
		"stripe-billing":     "pkg/stripe/stripe.go",
		"ai-inference":       "pkg/ai/stream.go",
		"vector-search":      "pkg/vector/vector.go",
		"rbac":               "pkg/rbac/rbac.go",
		"auth-oauth2":        "pkg/oauth2/oauth2.go",
		"otp-2fa":            "pkg/otp/otp.go",
		"api-keys":           "pkg/apikeys/apikeys.go",
		"s3-storage":         "pkg/storage/s3.go",
		"notifications":      "pkg/notifications/sse.go",
		"email-templates":    "pkg/email/email.go",
		"webhooks-engine":    "pkg/webhooks/signer.go",
		"audit-log":          "pkg/audit/audit.go",
		"multi-tenant":       "pkg/tenant/tenant.go",
		"feature-flags":      "pkg/flags/flags.go",
		"queues-asynq":       "pkg/queues/queues.go",
		"search-meilisearch": "pkg/search/search.go",
		"rate-limiter-redis": "pkg/ratelimit/limiter.go",
		"caching-redis":      "pkg/cache/cache.go",
		"cron-scheduler":     "pkg/cron/cron.go",
		"comments-reactions": "pkg/comments/tree.go",
		"activity-feed":      "pkg/feed/feed.go",
		"analytics-events":   "pkg/analytics/events.go",
	}

	if hFile, ok := helperPkgMap[mod.Name]; ok {
		allFiles = append(allFiles, hFile)
		if !dryRun {
			pkgDir := filepath.Dir(filepath.Join(targetDir, hFile))
			_ = os.MkdirAll(pkgDir, 0o755)
			// Find matching template constant
			content := getHelperContent(mod.Name)
			if content != "" {
				_ = os.WriteFile(filepath.Join(targetDir, hFile), []byte(content), 0o644)
			}
		}
	}

	return added, allFiles, nil
}

func getHelperContent(modName string) string {
	switch modName {
	case "stripe-billing":
		return StripeHelperTemplate
	case "ai-inference":
		return AIStreamHelperTemplate
	case "vector-search":
		return VectorHelperTemplate
	case "rbac":
		return RBACHelperTemplate
	case "auth-oauth2":
		return OAuth2HelperTemplate
	case "otp-2fa":
		return OTP2FAHelperTemplate
	case "api-keys":
		return APIKeysHelperTemplate
	case "s3-storage":
		return S3StorageHelperTemplate
	case "notifications":
		return NotificationsHelperTemplate
	case "email-templates":
		return EmailHelperTemplate
	case "webhooks-engine":
		return WebhooksHelperTemplate
	case "audit-log":
		return AuditLogHelperTemplate
	case "multi-tenant":
		return MultiTenantHelperTemplate
	case "feature-flags":
		return FeatureFlagsHelperTemplate
	case "queues-asynq":
		return QueuesHelperTemplate
	case "search-meilisearch":
		return SearchHelperTemplate
	case "rate-limiter-redis":
		return RateLimiterHelperTemplate
	case "caching-redis":
		return CacheHelperTemplate
	case "cron-scheduler":
		return CronHelperTemplate
	case "comments-reactions":
		return CommentsHelperTemplate
	case "activity-feed":
		return ActivityFeedHelperTemplate
	case "analytics-events":
		return AnalyticsHelperTemplate
	}
	return ""
}
