package util

import (
	"log"
	"os"
)

func GetEnvOrDefault(envVarName, defaultValue string) string {
	value, ok := os.LookupEnv(envVarName)
	if !ok {
		return defaultValue
	}

	return value
}

func MustGetEnv(envVarName string) string {
	value, ok := os.LookupEnv(envVarName)
	if ok {
		return value
	}
	log.Fatalf("environment variable %q is not set", envVarName)

	return ""
}
