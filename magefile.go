//go:build mage

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	packageName         = "github.com/dunamismax/go-web-server"
	binaryName          = "server"
	buildDir            = "bin"
	tmpDir              = "tmp"
	templVersion        = "v0.3.1001"
	sqlcVersion         = "v1.30.0"
	golangciLintVersion = "v2.11.3"
)

// Default target to run when none is specified
var Default = Build

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() error {
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// .env file doesn't exist, that's okay
		return nil
	}

	file, err := os.Open(envFile)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
				(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
				value = value[1 : len(value)-1]
			}

			// Only set if not already set by system environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}

	return scanner.Err()
}

// Build generates code and builds the server binary
func Build() error {
	mg.SerialDeps(Generate, buildServer)
	return nil
}

func buildServer() error {
	fmt.Println("Building server...")

	if err := sh.Run("mkdir", "-p", buildDir); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	ldflags := "-s -w -X main.version=1.0.0 -X main.buildTime=" + getCurrentTime()
	binaryPath := filepath.Join(buildDir, binaryName)

	// Add .exe extension on Windows
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	return sh.RunV("go", "build", "-ldflags="+ldflags, "-o", binaryPath, "./cmd/web")
}

func getCurrentTime() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func installGoTool(name, pkg string) error {
	fmt.Printf("  Installing %s...\n", name)
	if err := sh.RunV("go", "install", pkg); err != nil {
		return fmt.Errorf("failed to install %s: %w", name, err)
	}

	return nil
}

// getGoBinaryPath finds the path to a Go binary, checking GOBIN, GOPATH/bin, and PATH
func getGoBinaryPath(binaryName string) (string, error) {
	// First check if it's in PATH
	if err := sh.Run("which", binaryName); err == nil {
		return binaryName, nil
	}

	// Check GOBIN first
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		binaryPath := filepath.Join(gobin, binaryName)
		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath, nil
		}
	}

	// Check GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		if home := os.Getenv("HOME"); home != "" {
			gopath = filepath.Join(home, "go")
		}
	}

	if gopath != "" {
		binaryPath := filepath.Join(gopath, "bin", binaryName)
		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("%s not found in PATH, GOBIN, or GOPATH/bin", binaryName)
}

// Generate runs all code generation
func Generate() error {
	fmt.Println("Generating code...")
	mg.Deps(generateSqlc, generateTempl, buildCSS)
	return nil
}

func generateSqlc() error {
	fmt.Println("  Generating sqlc code...")
	sqlcPath, err := getGoBinaryPath("sqlc")
	if err != nil {
		return fmt.Errorf("sqlc not found: %w", err)
	}
	return sh.RunV(sqlcPath, "generate")
}

func generateTempl() error {
	fmt.Println("  Generating templ code...")
	templPath, err := getGoBinaryPath("templ")
	if err != nil {
		return fmt.Errorf("templ not found: %w", err)
	}
	return sh.RunV(templPath, "generate")
}

func buildCSS() error {
	fmt.Println("  Building Tailwind CSS...")

	// Check if npm is available
	if err := sh.Run("which", "npm"); err != nil {
		fmt.Println("    Warning: npm not found, skipping CSS build")
		return nil
	}

	// Check if node_modules exists
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		npmCommand := "install"
		if _, err := os.Stat("package-lock.json"); err == nil {
			npmCommand = "ci"
		}

		fmt.Printf("    Installing npm dependencies with npm %s...\n", npmCommand)
		if err := sh.RunV("npm", npmCommand); err != nil {
			return fmt.Errorf("failed to install npm dependencies with npm %s: %w", npmCommand, err)
		}
	}

	// Build CSS
	return sh.RunV("npm", "run", "build-css")
}

// Fmt formats and tidies code using goimports and standard tooling
func Fmt() error {
	fmt.Println("Formatting and tidying...")

	// Tidy go modules
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("failed to tidy modules: %w", err)
	}

	// Use goimports for better import management and formatting
	fmt.Println("  Running goimports...")
	goimportsPath, err := getGoBinaryPath("goimports")
	if err != nil {
		fmt.Printf("Warning: goimports not found, falling back to go fmt: %v\n", err)
		if err := sh.RunV("go", "fmt", "./..."); err != nil {
			return fmt.Errorf("failed to format code: %w", err)
		}
	} else {
		if err := sh.RunV(goimportsPath, "-w", "."); err != nil {
			fmt.Printf("Warning: goimports failed, falling back to go fmt: %v\n", err)
			if err := sh.RunV("go", "fmt", "./..."); err != nil {
				return fmt.Errorf("failed to format code: %w", err)
			}
		}
	}

	// Format templ files if templ is available
	if templPath, err := getGoBinaryPath("templ"); err == nil {
		fmt.Println("  Formatting templ files...")
		if err := sh.RunV(templPath, "fmt", "."); err != nil {
			fmt.Printf("Warning: failed to format templ files: %v\n", err)
		}
	}

	return nil
}

// Vet analyzes code for common errors
func Vet() error {
	fmt.Println("Running go vet...")
	return sh.RunV("go", "vet", "./...")
}

// Test runs the Go test suite
func Test() error {
	fmt.Println("Running go test...")
	return sh.RunV("go", "test", "./...")
}

// VulnCheck scans for known vulnerabilities
func VulnCheck() error {
	fmt.Println("Running vulnerability check...")
	govulncheckPath, err := getGoBinaryPath("govulncheck")
	if err != nil {
		return fmt.Errorf("govulncheck not found: %w", err)
	}
	return sh.RunV(govulncheckPath, "./...")
}

// Lint runs golangci-lint with comprehensive linting rules
func Lint() error {
	fmt.Println("Running golangci-lint...")

	// Match the CI version locally to avoid version-specific lint drift.
	if err := installGoTool("golangci-lint", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"+golangciLintVersion); err != nil {
		return fmt.Errorf("failed to install golangci-lint v2: %w", err)
	}

	// Find golangci-lint binary
	lintPath, err := getGoBinaryPath("golangci-lint")
	if err != nil {
		return fmt.Errorf("golangci-lint not found after installation: %w", err)
	}

	return sh.RunV(lintPath, "run", "./...")
}

// Run builds and runs the server
func Run() error {
	mg.SerialDeps(Build)
	fmt.Println("Starting server...")

	binaryPath := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	return sh.RunV(binaryPath)
}

// Dev starts development server with hot reload
func Dev() error {
	fmt.Println("Starting development server with hot reload...")

	// Find air binary
	airPath, err := getGoBinaryPath("air")
	if err != nil {
		if err := installGoTool("air", "github.com/air-verse/air@latest"); err != nil {
			return fmt.Errorf("failed to install air: %w", err)
		}
		// Try to find it again after installation
		airPath, err = getGoBinaryPath("air")
		if err != nil {
			return fmt.Errorf("air not found after installation: %w", err)
		}
	}

	return sh.RunV(airPath)
}

// Clean removes built binaries and generated files
func Clean() error {
	fmt.Println("Cleaning up...")

	// Remove build directory
	if err := sh.Rm(buildDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove build directory: %w", err)
	}

	// Remove tmp directory
	if err := sh.Rm(tmpDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove tmp directory: %w", err)
	}

	// Remove generated CSS
	if err := sh.Rm("internal/ui/static/css/styles.css"); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove generated CSS: %v\n", err)
	}
	if err := sh.Rm("internal/ui/static/css/pico.min.css"); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove generated Pico CSS: %v\n", err)
	}

	fmt.Println("Clean complete!")
	return nil
}

// Reset completely resets the repository to a fresh state
func Reset() error {
	fmt.Println("Resetting repository to clean state...")

	// First run clean to remove built artifacts
	if err := Clean(); err != nil {
		return fmt.Errorf("failed to clean build artifacts: %w", err)
	}

	// Reset database to fresh state
	fmt.Println("Resetting database...")
	// Note: Database reset now requires manual intervention for local PostgreSQL

	// Remove legacy SQLite database file if it exists
	if err := sh.Rm("data.db"); err == nil {
		fmt.Println("  Removed legacy SQLite database file")
	}

	// Remove any generated code to ensure fresh generation
	fmt.Println("Removing generated files...")
	generatedFiles := []string{
		"internal/view/home_templ.go",
		"internal/view/users_templ.go",
		"internal/view/layout/base_templ.go",
		"internal/store/queries.sql.go",
	}

	for _, file := range generatedFiles {
		if err := sh.Rm(file); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove %s: %v\n", file, err)
		}
	}

	// Regenerate code and run migrations to get fresh database with new sample data
	fmt.Println("Regenerating code and database...")
	if err := Generate(); err != nil {
		return fmt.Errorf("failed to regenerate code: %w", err)
	}

	if err := Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("Reset complete! Repository is now in fresh state with latest sample data.")
	fmt.Println("You can now run 'mage dev' or 'mage run' to see the changes.")
	return nil
}

// Setup installs required development tools
func Setup() error {
	fmt.Println("Setting up development environment...")

	tools := map[string]string{
		"templ":       "github.com/a-h/templ/cmd/templ@" + templVersion,
		"sqlc":        "github.com/sqlc-dev/sqlc/cmd/sqlc@" + sqlcVersion,
		"govulncheck": "golang.org/x/vuln/cmd/govulncheck@latest",
		"air":         "github.com/air-verse/air@latest",

		"goimports": "golang.org/x/tools/cmd/goimports@latest",
	}

	for tool, pkg := range tools {
		if err := installGoTool(tool, pkg); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}

	// Download module dependencies
	fmt.Println("Downloading dependencies...")
	if err := sh.RunV("go", "mod", "download"); err != nil {
		return fmt.Errorf("failed to download dependencies: %w", err)
	}

	fmt.Println("Setup complete!")
	fmt.Println("Next steps:")
	fmt.Println("   • Ensure PostgreSQL is running locally")
	fmt.Println("   • Run 'mage dev' to start development with hot reload")
	fmt.Println("   • Run 'mage build' to create production binary")

	return nil
}

// buildDatabaseURL constructs database URL from environment variables or uses default
func buildDatabaseURL() string {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		return databaseURL
	}

	fmt.Println("  Using local PostgreSQL...")
	// Build database URL from individual environment variables
	user := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	name := os.Getenv("DATABASE_NAME")
	sslmode := os.Getenv("DATABASE_SSLMODE")

	// Set defaults if not specified
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if name == "" {
		name = "gowebserver"
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	if user == "" || password == "" {
		fmt.Println("  Warning: DATABASE_USER and DATABASE_PASSWORD must be set in .env file")
		return "postgres://user:password@localhost:5432/gowebserver?sslmode=disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
}

// Migrate runs database migrations up using Atlas
func Migrate() error {
	fmt.Println("Running database migrations with Atlas...")

	// Load environment variables from .env file
	if err := loadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	// Check if atlas is installed
	if err := sh.Run("which", "atlas"); err != nil {
		fmt.Println("Atlas not found, please install it:")
		fmt.Println("  curl -sSf https://atlasgo.sh | sh")
		fmt.Println("  or")
		fmt.Println("  brew install ariga/tap/atlas")
		return fmt.Errorf("atlas CLI not found")
	}

	databaseURL := buildDatabaseURL()
	os.Setenv("DATABASE_URL", databaseURL)

	return sh.RunV("atlas", "migrate", "apply", "--env", "dev")
}

// MigrateDown shows Atlas migration status (Atlas doesn't support automatic rollbacks)
func MigrateDown() error {
	fmt.Println("Atlas doesn't support automatic rollbacks like Goose.")
	fmt.Println("To rollback, create a new migration that reverses the changes.")
	fmt.Println("Use 'mage migratestatus' to see current migration state.")
	return nil
}

// MigrateStatus shows migration status using Atlas
func MigrateStatus() error {
	fmt.Println("Checking migration status with Atlas...")

	// Load environment variables from .env file
	if err := loadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	// Check if atlas is installed
	if err := sh.Run("which", "atlas"); err != nil {
		fmt.Println("Atlas not found, please install it:")
		fmt.Println("  curl -sSf https://atlasgo.sh | sh")
		return fmt.Errorf("atlas CLI not found")
	}

	databaseURL := buildDatabaseURL()
	os.Setenv("DATABASE_URL", databaseURL)

	return sh.RunV("atlas", "migrate", "status", "--env", "dev")
}

// CI runs the complete non-mutating local CI pipeline
func CI() error {
	fmt.Println("Running complete CI pipeline...")
	mg.SerialDeps(Generate, Vet, Test, Lint, VulnCheck, buildServer, showBuildInfo)
	return nil
}

// Quality runs the main quality checks
func Quality() error {
	fmt.Println("Running all quality checks...")
	mg.Deps(Vet, Test, Lint, VulnCheck)
	return nil
}

// Release builds and creates a release using GoReleaser
func Release() error {
	fmt.Println("Creating release with GoReleaser...")

	// Check if goreleaser is installed
	if err := sh.Run("which", "goreleaser"); err != nil {
		fmt.Println("GoReleaser not found, please install it:")
		fmt.Println("  go install github.com/goreleaser/goreleaser/v2@latest")
		fmt.Println("  or")
		fmt.Println("  brew install goreleaser")
		return fmt.Errorf("goreleaser CLI not found")
	}

	return sh.RunV("goreleaser", "release", "--clean")
}

// Snapshot builds a snapshot release using GoReleaser (no publishing)
func Snapshot() error {
	fmt.Println("Creating snapshot release with GoReleaser...")

	// Check if goreleaser is installed
	if err := sh.Run("which", "goreleaser"); err != nil {
		fmt.Println("GoReleaser not found, please install it:")
		fmt.Println("  go install github.com/goreleaser/goreleaser/v2@latest")
		return fmt.Errorf("goreleaser CLI not found")
	}

	return sh.RunV("goreleaser", "build", "--snapshot", "--clean")
}

// Help prints a help message with available commands
func Help() {
	fmt.Println(`
Go Web Server Magefile

Available commands:

Development:
  mage setup (s)        Install all development tools and dependencies
  mage generate (g)     Generate sqlc and templ code
  mage dev (d)          Start development server with hot reload
  mage run (r)          Build and run server
  mage build (b)        Build production binary

Database:
  mage migrate (m)      Run database migrations up
  mage migrateDown      Roll back last migration
  mage migrateStatus    Show migration status

Quality:
  mage fmt (f)          Format code with goimports and tidy modules
  mage vet (v)          Run go vet static analysis
  mage test (t)         Run go test ./...
  mage lint (l)         Run golangci-lint comprehensive linting
  mage vulncheck (vc)   Check for security vulnerabilities
  mage quality (q)      Run main quality checks (vet + test + lint + vulncheck)

Release:
  mage release          Create and publish release using GoReleaser
  mage snapshot         Build snapshot release using GoReleaser (no publishing)

Production:
  mage ci               Complete CI pipeline (generate + vet + test + lint + vulncheck + build)
  mage clean (c)        Clean build artifacts and temporary files
  mage reset            Reset repository to fresh state (clean + reset database + regenerate)

Other:
  mage help (h)         Show this help message
	`)
}

// showBuildInfo displays information about the built binary
func showBuildInfo() error {
	binaryPath := filepath.Join(buildDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", binaryPath)
	}

	fmt.Println("\nBuild Information:")

	// Show binary size
	if info, err := os.Stat(binaryPath); err == nil {
		size := info.Size()
		fmt.Printf("   Binary size: %.2f MB\n", float64(size)/1024/1024)
	}

	// Show Go version
	if version, err := sh.Output("go", "version"); err == nil {
		fmt.Printf("   Go version: %s\n", version)
	}

	return nil
}

// Aliases for common commands
var Aliases = map[string]interface{}{
	"b":  Build,
	"g":  Generate,
	"f":  Fmt,
	"v":  Vet,
	"t":  Test,
	"l":  Lint,
	"vc": VulnCheck,
	"r":  Run,
	"d":  Dev,
	"c":  Clean,
	"s":  Setup,
	"q":  Quality,
	"m":  Migrate,
	"h":  Help,
}
