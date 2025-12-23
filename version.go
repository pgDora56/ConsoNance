package main

import "fmt"

const (
	// Version is the current version of ConsoNance
	Version = "25.12.1"
	
	// AppName is the application name
	AppName = "ConsoNance"
)

// GetVersionString returns a formatted version string
func GetVersionString() string {
	return fmt.Sprintf("%s v%s", AppName, Version)
}

