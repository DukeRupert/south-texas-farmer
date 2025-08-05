package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/antonlindstrom/pgstore"
	"github.com/dukerupert/south-texas-farmer/internal/auth"
	"github.com/dukerupert/south-texas-farmer/internal/database"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
)

// InitialUserConfig holds the configuration for creating initial users
type InitialUserConfig struct {
	Username  string
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type DatabaseConfig struct {
	PostgresDB       string
	PostgresHost     string
	PostgresUser     string
	PostgresPassword string
	PostgresPort     string
	PostgresSSL      string
}

type ClientConfig struct {
	Environment   string
	Port          string
	SessionSecret string
	Database      DatabaseConfig
	Admin         InitialUserConfig
}

func loadConfig() (*ClientConfig, error) {
	// Enable automatic environment variable reading
	viper.AutomaticEnv()

	// Set the config file name (without extension) and type
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	// Add the path where the .env file is located
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "1234")
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_SSL", "disable")
	viper.SetDefault("POSTGRES_PORT", "5432")
	viper.SetDefault("SESSION_SECRET", "supersecret")

	// Bind environment variables
	viper.BindEnv("APP_ENV")
	viper.BindEnv("APP_PORT")
	viper.BindEnv("POSTGRES_HOST")
	viper.BindEnv("POSTGRES_DB")
	viper.BindEnv("POSTGRES_USER")
	viper.BindEnv("POSTGRES_PASSWORD")
	viper.BindEnv("POSTGRES_PORT")
	viper.BindEnv("POSTGRES_SSL")
	viper.BindEnv("SESSION_SECRET")
	viper.BindEnv("ADMIN_USERNAME")
	viper.BindEnv("ADMIN_EMAIL")
	viper.BindEnv("ADMIN_PASSWORD")
	viper.BindEnv("ADMIN_FIRST_NAME")
	viper.BindEnv("ADMIN_LAST_NAME")

	// Read the configuration file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("No .env file found, relying on environment variables or defaults.")
		} else {
			log.Fatalf("Error reading config file: %s", err)
		}
	}

	database := &DatabaseConfig{
		PostgresHost:     viper.GetString("POSTGRES_HOST"),
		PostgresDB:       viper.GetString("POSTGRES_DB"),
		PostgresUser:     viper.GetString("POSTGRES_USER"),
		PostgresPassword: viper.GetString("POSTGRES_PASSWORD"),
		PostgresPort:     viper.GetString("POSTGRES_PORT"),
		PostgresSSL:      viper.GetString("POSTGRES_SSL"),
	}

	admin := &InitialUserConfig{
		Username:  viper.GetString("ADMIN_USERNAME"),
		Email:     viper.GetString("ADMIN_EMAIL"),
		Password:  viper.GetString("ADMIN_PASSWORD"),
		FirstName: viper.GetString("ADMIN_FIRST_NAME"),
		LastName:  viper.GetString("ADMIN_LAST_NAME"),
	}

	// Create and populate the config struct using the correct keys
	config := &ClientConfig{
		Environment:   viper.GetString("APP_ENV"),
		Port:          viper.GetString("APP_PORT"),
		SessionSecret: viper.GetString("SESSION_SECRET"),
		Database:      *database,
		Admin:         *admin,
	}

	return config, nil
}

// BuildPostgreSQLConnectionString creates a PostgreSQL connection string with SSL mode option
func BuildPostgreSQLConnectionString(host, database, user, password, port, sslMode string) string {
	encodedPassword := url.QueryEscape(password)

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, encodedPassword, host, port, database, sslMode)
}

// InitializeUsers creates initial users if enabled and if no users exist
func InitializeUser(ctx context.Context, queries *database.Queries, cfg InitialUserConfig) error {

	// Check if any users already exist
	userCount, err := queries.CountActiveUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count existing users: %w", err)
	}

	if userCount > 0 {
		log.Printf("Users already exist (%d found), skipping initialization", userCount)
		return nil
	}

	log.Println("No users found, initializing users from environment variables...")

	// Check config for user details

	if cfg.Email == "" {
		log.Println("No user email configured in environment variables")
		return nil
	}

	if cfg.Password == "" {
		log.Println("No user password in environment variables")
		return nil
	}

	// Create each user
	hashedPassword, err := auth.HashPassword(cfg.Password)
	if err != nil {
		slog.Error("failed to hash initial user password", slog.Any("error", err))
	}
	adminUser, err := queries.CreateUser(ctx, database.CreateUserParams{
		Username:     cfg.Username,
		Email:        cfg.Email,
		PasswordHash: hashedPassword,
		FirstName:    database.StringToPgText(cfg.FirstName),
		LastName:     database.StringToPgText(cfg.LastName),
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	log.Printf("Successfully created user: %s (%s)", adminUser.Username, adminUser.Email)

	return nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
	}
	connectionString := BuildPostgreSQLConnectionString(cfg.Database.PostgresHost, cfg.Database.PostgresDB, cfg.Database.PostgresUser, cfg.Database.PostgresPassword, cfg.Database.PostgresPort, cfg.Database.PostgresSSL)

	// Initialize database
	db, err := database.NewDB(connectionString)
	if err != nil {
		slog.Error("database connection failed", slog.Any("error", err))
	}
	defer db.Close()

	// Run migrations
	autoMigrate := cfg.Environment == "development"
	if err := db.RunMigrations(autoMigrate); err != nil {
		slog.Error("migrations failed", slog.Any("error", err))
	}

	// Initialize admin user
	err = InitializeUser(context.Background(), db.Queries, cfg.Admin)
	if err != nil {
		slog.Error("failed to create initial user", slog.Any("error", err))
	}

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Session middleware - configure with your session store
	store, err := pgstore.NewPGStore(connectionString, []byte(cfg.SessionSecret))
	if err != nil {
		slog.Error("failed to create session store", slog.Any("error", err))
	}
	defer store.Close()
	e.Use(session.Middleware(store))

	// Initialize services
	authService := auth.NewAuthService(db.Queries)
	authHandlers := auth.NewAuthHandlers(authService)

	// Public routes (guests only)
	guest := e.Group("", auth.GuestOnlyMiddleware())
	guest.GET("/login", authHandlers.ShowLogin)
	guest.POST("/login", authHandlers.Login)

	// Public routes (no restrictions)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome! Go to /login to authenticate.")
	})

	// Protected routes
	protected := e.Group("", auth.AuthMiddleware())
	protected.GET("/dashboard", auth.Dashboard)
	protected.POST("/logout", authHandlers.Logout)

	// API routes (protected)
	api := e.Group("/api", auth.AuthMiddleware())
	api.GET("/profile", func(c echo.Context) error {
		user, err := auth.GetCurrentUser(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get user info",
			})
		}
		return c.JSON(http.StatusOK, user)
	})

	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
