package hash

import (
	"time"

	"github.com/google/uuid"
)

// GenerateHash возвращает случайный "хэш"
func GenerateHash() string {
	time.Sleep(time.Second * 5) // имитируем долгую работу
	id := uuid.New()

	return id.String()
}
