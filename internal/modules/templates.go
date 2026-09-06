package modules

// StripeHelperTemplate is a complete production helper for Stripe API session management and webhooks.
const StripeHelperTemplate = `package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// StripeEvent represents a generic Stripe webhook event payload.
type StripeEvent struct {
	ID       string          ` + "`" + `json:"id"` + "`" + `
	Type     string          ` + "`" + `json:"type"` + "`" + `
	Data     StripeEventData ` + "`" + `json:"data"` + "`" + `
	Created  int64           ` + "`" + `json:"created"` + "`" + `
}

type StripeEventData struct {
	Object map[string]interface{} ` + "`" + `json:"object"` + "`" + `
}

// VerifyWebhookSignature checks timestamp tolerance and HMAC-SHA256 signature.
func VerifyWebhookSignature(payload []byte, sigHeader string, secret string, toleranceSeconds int64) (*StripeEvent, error) {
	if secret == "" {
		return nil, errors.New("stripe webhook secret is empty")
	}
	if sigHeader == "" {
		return nil, errors.New("missing Stripe-Signature header")
	}

	pairs := strings.Split(sigHeader, ",")
	var timestampStr, signature string
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			if parts[0] == "t" {
				timestampStr = parts[1]
			} else if parts[0] == "v1" {
				signature = parts[1]
			}
		}
	}

	if timestampStr == "" || signature == "" {
		return nil, errors.New("invalid Stripe-Signature header format")
	}

	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in signature: %w", err)
	}

	if toleranceSeconds > 0 {
		diff := time.Now().Unix() - ts
		if diff < 0 {
			diff = -diff
		}
		if diff > toleranceSeconds {
			return nil, fmt.Errorf("webhook timestamp outside tolerance (%d seconds)", diff)
		}
	}

	signedPayload := fmt.Sprintf("%s.%s", timestampStr, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
		return nil, errors.New("stripe signature verification failed")
	}

	var event StripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse stripe event JSON: %w", err)
	}

	return &event, nil
}

// GetStripeSecret returns the configured Stripe secret key from environment.
func GetStripeSecret() string {
	return os.Getenv("STRIPE_SECRET_KEY")
}
`

// AIStreamHelperTemplate is a complete production helper for Server-Sent Events (SSE) LLM token streaming.
const AIStreamHelperTemplate = `package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// StreamChunk represents an OpenAI-compatible SSE streaming response chunk.
type StreamChunk struct {
	ID      string         ` + "`" + `json:"id"` + "`" + `
	Object  string         ` + "`" + `json:"object"` + "`" + `
	Created int64          ` + "`" + `json:"created"` + "`" + `
	Model   string         ` + "`" + `json:"model"` + "`" + `
	Choices []StreamChoice ` + "`" + `json:"choices"` + "`" + `
}

type StreamChoice struct {
	Index        int         ` + "`" + `json:"index"` + "`" + `
	Delta        StreamDelta ` + "`" + `json:"delta"` + "`" + `
	FinishReason *string     ` + "`" + `json:"finish_reason"` + "`" + `
}

type StreamDelta struct {
	Role    string ` + "`" + `json:"role,omitempty"` + "`" + `
	Content string ` + "`" + `json:"content,omitempty"` + "`" + `
}

// WriteSSEFrame formats and flushes a single SSE data frame to a stream writer.
func WriteSSEFrame(w *bufio.Writer, event string, data any) error {
	if event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
			return err
		}
	}

	var payload []byte
	var err error
	switch v := data.(type) {
	case string:
		payload = []byte(v)
	case []byte:
		payload = v
	default:
		payload, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(payload)); err != nil {
		return err
	}
	return w.Flush()
}

// WriteDoneFrame emits the standard OpenAI SSE completion token: data: [DONE].
func WriteDoneFrame(w *bufio.Writer) error {
	_, err := fmt.Fprintf(w, "data: [DONE]\n\n")
	if err != nil {
		return err
	}
	return w.Flush()
}
`

// VectorHelperTemplate provides complete vector distance and similarity functions for pgvector / embeddings.
const VectorHelperTemplate = `package vector

import (
	"errors"
	"fmt"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two float vectors.
func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimension mismatch: %d vs %d", len(a), len(b))
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0, nil
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}

// EuclideanDistance calculates L2 distance between two vectors.
func EuclideanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimension mismatch: %d vs %d", len(a), len(b))
	}
	var sum float64
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum), nil
}

// FormatPgVector formats a float vector as a Postgres pgvector string literal e.g. "[0.1, 0.2, 0.3]".
func FormatPgVector(vec []float64) string {
	strs := make([]string, len(vec))
	for i, v := range vec {
		strs[i] = fmt.Sprintf("%f", v)
	}
	return "[" + stringsJoin(strs, ",") + "]"
}

func stringsJoin(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	res := elems[0]
	for _, s := range elems[1:] {
		res += sep + s
	}
	return res
}
`

// RBACHelperTemplate provides complete Role-Based Access Control middleware and permission evaluation.
const RBACHelperTemplate = `package rbac

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Evaluator checks permissions with wildcard matching (e.g. "posts:*", "*").
type Evaluator struct{}

func (e *Evaluator) HasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == "*" || perm == requiredPermission {
			return true
		}
		if strings.HasSuffix(perm, ":*") {
			prefix := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(requiredPermission, prefix+":") {
				return true
			}
		}
	}
	return false
}

// RequirePermission creates a Fiber middleware asserting the request user has the required permission.
func RequirePermission(requiredPermission string) fiber.Handler {
	eval := &Evaluator{}
	return func(c *fiber.Ctx) error {
		perms, ok := c.Locals("permissions").([]string)
		if !ok || len(perms) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: user has no assigned permissions",
			})
		}

		if !eval.HasPermission(perms, requiredPermission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: missing required permission '" + requiredPermission + "'",
			})
		}

		return c.Next()
	}
}
`

// OAuth2HelperTemplate provides OAuth state generation and token exchange helper contracts.
const OAuth2HelperTemplate = `package oauth2

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GenerateStateToken creates a cryptographically secure random state parameter for OAuth2 CSRF prevention.
func GenerateStateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetGoogleConfig returns Google OAuth2 app credentials from environment variables.
func GetGoogleConfig() (*Config, error) {
	id := os.Getenv("GOOGLE_CLIENT_ID")
	secret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirect := os.Getenv("GOOGLE_REDIRECT_URL")
	if id == "" || secret == "" {
		return nil, errors.New("missing GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET")
	}
	return &Config{ClientID: id, ClientSecret: secret, RedirectURL: redirect}, nil
}
`

// OTP2FAHelperTemplate provides RFC 6238 TOTP 2FA secret generation and validation algorithm.
const OTP2FAHelperTemplate = `package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"time"
)

// GenerateSecret creates a 20-byte base32 TOTP secret key.
func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// GenerateCode calculates the 6-digit TOTP token for a given timestamp step.
func GenerateCode(secret string, timestamp time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	counter := uint64(timestamp.Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := (int32(hash[offset]&0x7f)<<24 |
		int32(hash[offset+1])<<16 |
		int32(hash[offset+2])<<8 |
		int32(hash[offset+3])) % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// BuildOTPAuthURL generates an otpauth:// URL suitable for rendering a QR code.
func BuildOTPAuthURL(issuer, accountName, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer), url.PathEscape(accountName), secret, url.QueryEscape(issuer))
}
`

// APIKeysHelperTemplate provides API key generation, constant-time comparison, and header authentication.
const APIKeysHelperTemplate = `package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GenerateKey produces a full key (ff_live_...), prefix, and SHA-256 hash.
func GenerateKey() (fullKey, prefix, hash string, err error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", err
	}
	randomHex := hex.EncodeToString(b)
	prefix = "ff_live_" + randomHex[:8]
	fullKey = prefix + "_" + randomHex[8:]

	h := sha256.Sum256([]byte(fullKey))
	hash = hex.EncodeToString(h[:])
	return fullKey, prefix, hash, nil
}

// VerifyKey performs constant-time hash comparison to prevent timing attacks.
func VerifyKey(providedKey string, storedHash string) bool {
	h := sha256.Sum256([]byte(providedKey))
	providedHash := hex.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(providedHash), []byte(storedHash)) == 1
}

// Middleware creates a Fiber handler that validates the X-API-Key header.
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing X-API-Key or Authorization header"})
		}
		c.Locals("raw_api_key", key)
		return c.Next()
	}
}
`

// S3StorageHelperTemplate provides S3 object key formatting and content-type helper.
const S3StorageHelperTemplate = `package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// BuildObjectKey generates a collision-resistant S3 object key with timestamp prefix.
func BuildObjectKey(folder, fileName string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	cleanBase := strings.ReplaceAll(strings.ToLower(base), " ", "-")
	return fmt.Sprintf("%s/%d_%s%s", strings.Trim(folder, "/"), time.Now().UnixNano(), cleanBase, ext)
}

// DetectMimeType returns a default content type by file extension.
func DetectMimeType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
`

// NotificationsHelperTemplate provides SSE & WebSocket notification channel hub.
const NotificationsHelperTemplate = `package notifications

import (
	"sync"
)

// ConnectionHub manages active user SSE notification channels.
type ConnectionHub struct {
	mu          sync.RWMutex
	userStreams map[string][]chan string
}

func NewConnectionHub() *ConnectionHub {
	return &ConnectionHub{
		userStreams: make(map[string][]chan string),
	}
}

func (h *ConnectionHub) Register(userID string) chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 10)
	h.userStreams[userID] = append(h.userStreams[userID], ch)
	return ch
}

func (h *ConnectionHub) BroadcastToUser(userID string, notificationJSON string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if streams, ok := h.userStreams[userID]; ok {
		for _, ch := range streams {
			select {
			case ch <- notificationJSON:
			default:
			}
		}
	}
}
`

// EmailHelperTemplate provides HTML template loading and SMTP transactional email sending.
const EmailHelperTemplate = `package email

import (
	"fmt"
	"net/smtp"
	"os"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func GetConfig() *Config {
	return &Config{
		Host:     getEnvOrDefault("SMTP_HOST", "smtp.mailtrap.io"),
		Port:     getEnvOrDefault("SMTP_PORT", "2525"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getEnvOrDefault("SMTP_FROM", "no-reply@fiberforge.dev"),
	}
}

func Send(to string, subject string, htmlBody string) error {
	cfg := GetConfig()
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		to, cfg.From, subject, htmlBody))

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
`

// WebhooksHelperTemplate provides HMAC signature signing and exponential backoff retry calculations.
const WebhooksHelperTemplate = `package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"time"
)

// SignPayload generates an HMAC-SHA256 signature for outgoing webhook HTTP headers.
func SignPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// CalculateBackoff returns exponential retry delay for a given attempt index (e.g. 1s, 4s, 16s, 64s).
func CalculateBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
`

// AuditLogHelperTemplate provides HTTP mutation auditing middleware.
const AuditLogHelperTemplate = `package audit

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type AuditEntry struct {
	Method    string ` + "`" + `json:"method"` + "`" + `
	Path      string ` + "`" + `json:"path"` + "`" + `
	IP        string ` + "`" + `json:"ip"` + "`" + `
	UserAgent string ` + "`" + `json:"user_agent"` + "`" + `
	Status    int    ` + "`" + `json:"status"` + "`" + `
	Duration  int64  ` + "`" + `json:"duration_ms"` + "`" + `
}

// Middleware creates a Fiber handler recording HTTP mutations (POST, PUT, DELETE, PATCH).
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		method := c.Method()

		if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
			entry := AuditEntry{
				Method:    method,
				Path:      c.Path(),
				IP:        c.IP(),
				UserAgent: c.Get("User-Agent"),
				Status:    c.Response().StatusCode(),
				Duration:  time.Since(start).Milliseconds(),
			}
			c.Locals("audit_entry", entry)
		}
		return err
	}
}
`

// MultiTenantHelperTemplate provides X-Tenant-ID resolution middleware.
const MultiTenantHelperTemplate = `package tenant

import (
	"github.com/gofiber/fiber/v2"
)

// RequireTenant extracts X-Tenant-ID header and ensures tenant context exists.
func RequireTenant() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Get("X-Tenant-ID")
		if tenantID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required X-Tenant-ID header",
			})
		}
		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
}
`

// FeatureFlagsHelperTemplate provides FNV-1a user rollout evaluation.
const FeatureFlagsHelperTemplate = `package flags

import (
	"hash/fnv"
)

// EvaluateRollout determines if a user falls within a feature flag's rollout percentage (0-100%).
func EvaluateRollout(userID string, percentage int) bool {
	if percentage >= 100 {
		return true
	}
	if percentage <= 0 {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	val := int(h.Sum32() % 100)
	return val < percentage
}
`

// QueuesHelperTemplate provides task payload serialization.
const QueuesHelperTemplate = `package queues

import (
	"encoding/json"
	"fmt"
)

type TaskPayload struct {
	Type    string          ` + "`" + `json:"type"` + "`" + `
	Data    json.RawMessage ` + "`" + `json:"data"` + "`" + `
	TraceID string          ` + "`" + `json:"trace_id"` + "`" + `
}

func NewTaskPayload(taskType string, data any, traceID string) (*TaskPayload, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task data: %w", err)
	}
	return &TaskPayload{Type: taskType, Data: bytes, TraceID: traceID}, nil
}
`

// SearchHelperTemplate provides query string cleaning and search parameter building.
const SearchHelperTemplate = `package search

import (
	"strings"
)

func CleanQuery(query string) string {
	q := strings.TrimSpace(query)
	q = strings.ReplaceAll(q, "'", "")
	q = strings.ReplaceAll(q, ";", "")
	return q
}
`

// RateLimiterHelperTemplate provides Redis rate limit sliding window key generation.
const RateLimiterHelperTemplate = `package ratelimit

import (
	"fmt"
	"time"
)

func BuildWindowKey(prefix string, identifier string, windowSeconds int) string {
	bucket := time.Now().Unix() / int64(windowSeconds)
	return fmt.Sprintf("ratelimit:%s:%s:%d", prefix, identifier, bucket)
}
`

// CacheHelperTemplate provides cache key generation.
const CacheHelperTemplate = `package cache

import (
	"fmt"
	"strings"
)

func BuildKey(namespace string, keys ...string) string {
	return fmt.Sprintf("cache:%s:%s", namespace, strings.Join(keys, ":"))
}
`

// CronHelperTemplate provides cron job execution logger.
const CronHelperTemplate = `package cron

import (
	"log/slog"
	"time"
)

func LogExecution(jobName string, start time.Time, err error) {
	dur := time.Since(start)
	if err != nil {
		slog.Error("cron job failed", "job", jobName, "duration_ms", dur.Milliseconds(), "error", err)
	} else {
		slog.Info("cron job completed successfully", "job", jobName, "duration_ms", dur.Milliseconds())
	}
}
`

// CommentsHelperTemplate provides recursive comment tree structure builder.
const CommentsHelperTemplate = `package comments

type CommentFlat struct {
	ID       string ` + "`" + `json:"id"` + "`" + `
	ParentID string ` + "`" + `json:"parent_id,omitempty"` + "`" + `
	Body     string ` + "`" + `json:"body"` + "`" + `
}

type CommentTree struct {
	ID       string         ` + "`" + `json:"id"` + "`" + `
	Body     string         ` + "`" + `json:"body"` + "`" + `
	Children []*CommentTree ` + "`" + `json:"children"` + "`" + `
}

func BuildTree(flatComments []CommentFlat) []*CommentTree {
	nodes := make(map[string]*CommentTree)
	var roots []*CommentTree

	for _, c := range flatComments {
		nodes[c.ID] = &CommentTree{ID: c.ID, Body: c.Body, Children: []*CommentTree{}}
	}

	for _, c := range flatComments {
		node := nodes[c.ID]
		if c.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodes[c.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	return roots
}
`

// ActivityFeedHelperTemplate provides activity story string formatting.
const ActivityFeedHelperTemplate = `package feed

import (
	"fmt"
)

func FormatStory(actorName, verb, targetTitle string) string {
	return fmt.Sprintf("%s %s %s", actorName, verb, targetTitle)
}
`

// AnalyticsHelperTemplate provides analytics payload parsing.
const AnalyticsHelperTemplate = `package analytics

type Event struct {
	Name      string         ` + "`" + `json:"name"` + "`" + `
	UserID    string         ` + "`" + `json:"user_id,omitempty"` + "`" + `
	Props     map[string]any ` + "`" + `json:"properties,omitempty"` + "`" + `
	Timestamp int64          ` + "`" + `json:"timestamp"` + "`" + `
}
`
