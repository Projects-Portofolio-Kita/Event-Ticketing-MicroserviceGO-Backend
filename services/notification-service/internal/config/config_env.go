// Package config loads runtime parameters and environment options for notification-service.
//
// File: config_env.go
// Usability: Provides helper functions to safely read string, integer, and boolean environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
)

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return defaultVal
	}
	valStr = strings.TrimSpace(strings.ToLower(valStr))
	if valStr == "true" || valStr == "1" || valStr == "yes" {
		return true
	}
	if valStr == "false" || valStr == "0" || valStr == "no" {
		return false
	}
	return defaultVal
}
