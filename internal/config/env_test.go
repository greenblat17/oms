package config

import (
	"os"
	"testing"
)

func TestGetter(t *testing.T) {
	firstTest, secondTest := "first test", "second test"
	envForTest := "ENV_FOR_TEST"

	env := GetValue(envForTest, firstTest)
	if env != firstTest {
		t.Fatal()
	}

	if err := os.Setenv(envForTest, secondTest); err != nil {
		env = GetValue(envForTest, firstTest)
		if env == firstTest {
			t.Fatal()
		}
	}
}
