package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/dukerupert/south-texas-farmer/internal/database"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Common errors for authentication
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
)

// AuthService handles authentication logic
type AuthService struct {
	db *database.Queries
}

func NewAuthService(db *database.Queries) *AuthService {
	return &AuthService{db: db}
}

func (a *AuthService) ValidateCredentials(ctx context.Context, username, password string) (*database.User, error) {
	// Get user from database
	user, err := a.db.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Don't reveal whether user exists or not
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Compare the provided password with stored hash
	if err := a.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, err // This will be ErrInvalidCredentials if password doesn't match
	}

	return &user, nil
}

// HashPassword creates a bcrypt hash from a plain text password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// ComparePassword compares a plain text password with a bcrypt hash
func (a *AuthService) ComparePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			// Return a generic error to avoid leaking information
			return ErrInvalidCredentials
		}
		return fmt.Errorf("password comparison failed: %w", err)
	}
	return nil
}

// Session constants
const (
	SessionName = "app-session"
	UserIDKey   = "user_id"
	UsernameKey = "username"
	IsAuthKey   = "authenticated"
)

// AuthMiddleware checks if user is authenticated
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get(SessionName, c)
			if err != nil {
				return c.Redirect(http.StatusFound, "/login")
			}

			// Check if user is authenticated
			authenticated, ok := sess.Values[IsAuthKey].(bool)
			if !ok || !authenticated {
				return c.Redirect(http.StatusFound, "/login")
			}

			// Optional: Add user info to context for easy access
			if userID, ok := sess.Values[UserIDKey].(int); ok {
				c.Set("user_id", userID)
			}
			if username, ok := sess.Values[UsernameKey].(string); ok {
				c.Set("username", username)
			}

			return next(c)
		}
	}
}

// Middleware to redirect authenticated users away from login/register
func GuestOnlyMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get(SessionName, c)
			if err == nil {
				if authenticated, ok := sess.Values[IsAuthKey].(bool); ok && authenticated {
					return c.Redirect(http.StatusFound, "/dashboard")
				}
			}
			return next(c)
		}
	}
}

// Auth handlers
type AuthHandlers struct {
	authService *AuthService
}

func NewAuthHandlers(authService *AuthService) *AuthHandlers {
	return &AuthHandlers{authService: authService}
}

// Login form (GET)
func (h *AuthHandlers) ShowLogin(c echo.Context) error {
	// In a real app, render your login template
	return c.HTML(http.StatusOK, `
		<form method="POST" action="/login">
			<input type="text" name="username" placeholder="Username" required>
			<input type="password" name="password" placeholder="Password" required>
			<button type="submit">Login</button>
		</form>
	`)
}

// Login handler (POST)
func (h *AuthHandlers) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username and password are required",
		})
	}

	// Validate credentials
	user, err := h.authService.ValidateCredentials(c.Request().Context(), username, password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid credentials",
		})
	}

	// Create session
	sess, err := session.Get(SessionName, c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create session",
		})
	}

	// Set session values
	sess.Values[IsAuthKey] = true
	sess.Values[UserIDKey] = user.ID
	sess.Values[UsernameKey] = user.Username

	// Configure session options
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}

	// Save session
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save session",
		})
	}

	return c.Redirect(http.StatusFound, "/dashboard")
}

// Logout handler
func (h *AuthHandlers) Logout(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Clear session values
	sess.Values[IsAuthKey] = false
	delete(sess.Values, UserIDKey)
	delete(sess.Values, UsernameKey)

	// Set MaxAge to -1 to delete the session
	sess.Options.MaxAge = -1

	// Save the session (this will delete it)
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, "/login")
}

// Protected route example
func Dashboard(c echo.Context) error {
	username := c.Get("username").(string)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Welcome to dashboard, %s!", username),
		"user_id": c.Get("user_id"),
	})
}

// Helper function to get current user from session
func GetCurrentUser(c echo.Context) (*database.User, error) {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		return nil, err
	}

	userID, ok := sess.Values[UserIDKey].(int)
	if !ok {
		return nil, fmt.Errorf("user not found in session")
	}

	username, ok := sess.Values[UsernameKey].(string)
	if !ok {
		return nil, fmt.Errorf("username not found in session")
	}

	return &database.User{
		ID:       int32(userID),
		Username: username,
	}, nil
}
