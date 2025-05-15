package main

import (
	"fmt"
	"os"

	"github.com/localrivet/projectmemory/internal/config"
)

func main() {
	// Set MCP_MODE environment variable
	os.Setenv("MCP_MODE", "1")

	fmt.Println("=== Starting MCP Mode Test ===")
	fmt.Println("This line should be visible in terminal output")

	// This should not print any log messages to stdout
	// because we're in MCP mode
	cfg, err := config.LoadConfigWithPath(".projectmemoryconfig")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Print to confirm config was loaded
	fmt.Println("=== Config Loaded Successfully ===")
	fmt.Printf("SQLite Path: %s\n", cfg.Store.SQLitePath)
	fmt.Printf("Log Level: %s\n", cfg.Logging.Level)

	// Reset MCP_MODE
	os.Setenv("MCP_MODE", "")

	fmt.Println("\n=== Testing Without MCP Mode ===")

	// This should print log messages to stdout
	// because we're not in MCP mode anymore
	cfg2, err := config.LoadConfigWithPath(".projectmemoryconfig")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Config Loaded Again ===")
	fmt.Printf("SQLite Path: %s\n", cfg2.Store.SQLitePath)
	fmt.Printf("Log Level: %s\n", cfg2.Logging.Level)

	fmt.Println("\n=== Test Complete ===")
}
