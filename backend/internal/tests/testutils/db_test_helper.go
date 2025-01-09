package testutils

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"simple-gin-backend/internal/config"
	"simple-gin-backend/internal/database"
	"simple-gin-backend/internal/models"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

// Create a new test DB, if it doesn't exist
func createTestDB() {
	// Connect to the default database (e.g., postgres) to create the test DB
	config.LoadConfig()
	defaultDSN := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.PostgresHost,
		config.AppConfig.PostgresUser,
		config.AppConfig.PostgresPassword,
		"postgres",
		config.AppConfig.PostgresPort,
	)

	db, err := sql.Open("postgres", defaultDSN)
	if err != nil {
		log.Fatalf("Failed to connect to default database: %v", err)
	}
	defer db.Close()

	// Test DB name
	testDBName := config.AppConfig.PostgresDb + "_test"

	// Terminate all connections to the test database
	_, err = db.Exec(fmt.Sprintf(`
		-- Terminate all connections to the test database
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid();
	`, testDBName))

	if err != nil {
		log.Fatalf("Failed to terminate connections to the test database %s: %v", testDBName, err)
	}

	// Drop the test database if it exists
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if err != nil {
		log.Fatalf("Failed to drop test database %s: %v", testDBName, err)
	}

	// Create the test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		log.Fatalf("Failed to create test database %s: %v", testDBName, err)
	} else {
		log.Printf("Database %s created successfully", testDBName)
	}
}

// TestDB holds the test database connection
var TestDB *gorm.DB

// SetupTestDatabase sets up the test database connection and runs migrations
func SetupTestDatabase() {
	config.LoadConfig()
	createTestDB()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.PostgresHost,
		config.AppConfig.PostgresUser,
		config.AppConfig.PostgresPassword,
		config.AppConfig.PostgresDb+"_test",
		config.AppConfig.PostgresPort,
	)

	var err error
	TestDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations (you can add your models here)
	TestDB.AutoMigrate(&models.Item{}, &models.User{})
}

// ResetTestDatabase resets the database after each test
func ResetTestDatabase() {
	TestDB.Exec("TRUNCATE TABLE items RESTART IDENTITY CASCADE")
}

// TearDownTestDatabase closes the database connection
func TearDownTestDatabase() {
	sqlDB, err := TestDB.DB()
	if err != nil {
		log.Fatalf("Failed to close the database: %v", err)
	}
	sqlDB.Close()
}

// PatchDatabase replaces the global database.DB with TestDB for testing
func PatchDatabase() {
	database.DB = TestDB
}

// UnpatchDatabase restores the original database.DB (if needed)
func UnpatchDatabase() {
	// Reset database.DB back to the development DB after testing
	database.InitDB()
}

// InitializeTestSuite sets up the test environment and patches the database connection
func InitializeTestSuite(m *testing.M) {
	// Initialize the test database
	SetupTestDatabase()

	// Patch the global database.DB to use the TestDB
	PatchDatabase()

	// Run all tests
	code := m.Run()

	// Teardown the database connection or other cleanup tasks
	TearDownTestDatabase()

	// Unpatch the database or clean up after tests
	UnpatchDatabase()

	// Exit with the test run status code
	os.Exit(code)
}
