package templates

// AuthTemplate renders the auth model, service, controllers, JWT utilities and
// middleware. It is only emitted when the auth feature is enabled.
const AuthTemplate = `// auth/password.go
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword compares a plaintext password with a bcrypt hash.
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
`

// JWTTemplate renders the JWT signing/verifying utilities.
const JWTTemplate = `// auth/jwt.go
package auth

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// claims is the JWT payload shape.
type claims struct {
	UserID uint ` + "`json:\"sub\"`" + `
	jwt.RegisteredClaims
}

// Secret returns the JWT signing secret from the environment.
func Secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	log.Fatal("JWT_SECRET environment variable is not set")
	return nil
}

// GenerateToken issues a signed token for the given user id.
func GenerateToken(userID uint) (string, error) {
	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(Secret())
}

// ValidateToken parses and verifies a token, returning the user id.
func ValidateToken(tokenStr string) (uint, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return Secret(), nil
	})
	if err != nil {
		return 0, err
	}
	if !token.Valid {
		return 0, errors.New("invalid token")
	}
	return c.UserID, nil
}
`

// MiddlewareTemplate renders the Fiber JWT middleware.
const MiddlewareTemplate = `// middleware/jwt.go
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/auth"
)

// JWT guards routes that require authentication.
func JWT(secret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "missing authorization header"}})
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "malformed authorization header"}})
		}
		uid, err := auth.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "invalid or expired token"}})
		}
		c.Locals("userId", uid)
		return c.Next()
	}
}
`

// AuthServiceTemplate renders the auth user store (SQL/GORM).
const AuthServiceTemplate = `// auth/store.go
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"{{.AppName}}/databases"
	"{{.AppName}}/model"
)

// Register creates a new user with a hashed password.
func Register(email, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Email: email, Password: string(hash)}
	if err := databases.DB.Create(user).Error; err != nil {
		return nil, errors.New("unable to register user: " + err.Error())
	}
	return user, nil
}

// Login verifies credentials and returns the user.
func Login(email, password string) (*model.User, error) {
	var user model.User
	if err := databases.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &user, nil
}
`

// MongoAuthServiceTemplate renders the auth user store (MongoDB/mgm).
const MongoAuthServiceTemplate = `// auth/store.go
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/kamva/mgm/v3"

	"{{.AppName}}/model"
)

// Register creates a new user with a hashed password.
func Register(email, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Email: email, Password: string(hash)}
	if err := mgm.Coll(&model.User{}).Create(user); err != nil {
		return nil, errors.New("unable to register user: " + err.Error())
	}
	return user, nil
}

// Login verifies credentials and returns the user.
func Login(email, password string) (*model.User, error) {
	filter := bson.M{"email": email}
	var user model.User
	if err := mgm.Coll(&model.User{}).First(filter, &user); err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &user, nil
}
`

// AuthUserServiceTemplate renders a GetUserByID helper for the user service (SQL).
const AuthUserServiceTemplate = `// service/user_service.go
package service

import (
	"errors"

	"{{.AppName}}/databases"
	"{{.AppName}}/model"
)

// GetUserByID returns a user by primary key without the password hash.
func GetUserByID(id uint) (*model.User, error) {
	var m model.User
	if err := databases.DB.First(&m, id).Error; err != nil {
		return nil, errors.New("user not found")
	}
	m.Password = ""
	return &m, nil
}
`

// MongoAuthUserServiceTemplate renders GetUserByID for the user service (Mongo).
const MongoAuthUserServiceTemplate = `// service/user_service.go
package service

import (
	"errors"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"{{.AppName}}/model"
)

// GetUserByID returns a user by id without the password hash.
func GetUserByID(id string) (*model.User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid id")
	}
	var m model.User
	if err := mgm.Coll(&model.User{}).FindByID(oid, &m); err != nil {
		return nil, errors.New("user not found")
	}
	m.Password = ""
	return &m, nil
}
`

// AuthControllerTemplate renders the register/login/refresh/me handlers (SQL).
const AuthControllerTemplate = `// controller/auth_controller.go
package controller

import (
	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/auth"
	"{{.AppName}}/service"
)

type credentials struct {
	Email    string ` + "`json:\"email\"`" + `
	Password string ` + "`json:\"password\"`" + `
}

// Register handles POST /api/auth/register.
func Register(c *fiber.Ctx) error {
	var req credentials
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&req); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	user, err := auth.Register(req.Email, req.Password)
	if err != nil {
		return SendError(c, fiber.StatusConflict, "CONFLICT", err.Error(), nil)
	}
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusCreated, fiber.Map{"token": token})
}

// Login handles POST /api/auth/login.
func Login(c *fiber.Ctx) error {
	var req credentials
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&req); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	user, err := auth.Login(req.Email, req.Password)
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
	}
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"token": token})
}

// Me handles GET /api/auth/me (protected).
func Me(c *fiber.Ctx) error {
	id, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "unauthenticated", nil)
	}
	m, err := service.GetUserByID(id)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Refresh re-issues a token for the authenticated user (protected).
func Refresh(c *fiber.Ctx) error {
	id, ok := c.Locals("userId").(uint)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "unauthenticated", nil)
	}
	token, err := auth.GenerateToken(id)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"token": token})
}
`

// MongoAuthControllerTemplate renders the auth handlers (MongoDB).
const MongoAuthControllerTemplate = `// controller/auth_controller.go
package controller

import (
	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/auth"
	"{{.AppName}}/service"
)

type credentials struct {
	Email    string ` + "`json:\"email\"`" + `
	Password string ` + "`json:\"password\"`" + `
}

// Register handles POST /api/auth/register.
func Register(c *fiber.Ctx) error {
	var req credentials
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&req); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	user, err := auth.Register(req.Email, req.Password)
	if err != nil {
		return SendError(c, fiber.StatusConflict, "CONFLICT", err.Error(), nil)
	}
	token, err := auth.GenerateToken(user.ID.Hex())
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusCreated, fiber.Map{"token": token})
}

// Login handles POST /api/auth/login.
func Login(c *fiber.Ctx) error {
	var req credentials
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}
	if errs := ValidateStruct(&req); len(errs) > 0 {
		return SendError(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "invalid request payload", errs)
	}
	user, err := auth.Login(req.Email, req.Password)
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
	}
	token, err := auth.GenerateToken(user.ID.Hex())
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"token": token})
}

// Me handles GET /api/auth/me (protected).
func Me(c *fiber.Ctx) error {
	id, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "unauthenticated", nil)
	}
	m, err := service.GetUserByID(id)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	}
	return SendSuccess(c, fiber.StatusOK, m)
}

// Refresh re-issues a token for the authenticated user (protected).
func Refresh(c *fiber.Ctx) error {
	id, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "unauthenticated", nil)
	}
	token, err := auth.GenerateToken(id)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token", nil)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{"token": token})
}
`

// MongoJWTTemplate renders JWT utilities using string (ObjectID) subjects.
const MongoJWTTemplate = `// auth/jwt.go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// claims is the JWT payload shape.
type claims struct {
	Subject string ` + "`json:\"sub\"`" + `
	jwt.RegisteredClaims
}

// Secret returns the JWT signing secret from the environment.
func Secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("change-me-in-production")
}

// GenerateToken issues a signed token for the given user id.
func GenerateToken(subject string) (string, error) {
	c := claims{
		Subject: subject,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(Secret())
}

// ValidateToken parses and verifies a token, returning the subject.
func ValidateToken(tokenStr string) (string, error) {
	var c claims
	token, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return Secret(), nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	return c.Subject, nil
}
`

// MongoMiddlewareTemplate renders the JWT middleware storing a string subject.
const MongoMiddlewareTemplate = `// middleware/jwt.go
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"{{.AppName}}/auth"
)

// JWT guards routes that require authentication.
func JWT(secret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "missing authorization header"}})
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "malformed authorization header"}})
		}
		sub, err := auth.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "UNAUTHORIZED", "message": "invalid or expired token"}})
		}
		c.Locals("userId", sub)
		return c.Next()
	}
}
`
