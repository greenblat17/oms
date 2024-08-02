package config

import "os"

// GetValue возвращает значение переменной окружения с указанным ключом.
// Если ключа нет - возвращает default value
func GetValue(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultValue
}
