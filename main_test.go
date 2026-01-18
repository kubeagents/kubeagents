package main

import (
	"testing"

	"github.com/kubeagents/kubeagents/store"
)

func TestInitJWTSecret_WithConfigSecret(t *testing.T) {
	st := store.NewMemoryStore()
	configSecret := "my-configured-secret"

	secret, err := initJWTSecret(st, configSecret)
	if err != nil {
		t.Fatalf("initJWTSecret() error = %v, want nil", err)
	}

	if secret != configSecret {
		t.Errorf("initJWTSecret() secret = %v, want %v", secret, configSecret)
	}

	// Verify it was saved to storage
	storedSecret, err := st.GetConfig(jwtSecretConfigKey)
	if err != nil {
		t.Fatalf("GetConfig() error = %v, want nil", err)
	}
	if storedSecret != configSecret {
		t.Errorf("stored secret = %v, want %v", storedSecret, configSecret)
	}
}

func TestInitJWTSecret_FromStorage(t *testing.T) {
	st := store.NewMemoryStore()
	existingSecret := "existing-secret-in-storage"

	// Pre-set a secret in storage
	err := st.SetConfig(jwtSecretConfigKey, existingSecret)
	if err != nil {
		t.Fatalf("SetConfig() error = %v, want nil", err)
	}

	// Call initJWTSecret without config secret
	secret, err := initJWTSecret(st, "")
	if err != nil {
		t.Fatalf("initJWTSecret() error = %v, want nil", err)
	}

	if secret != existingSecret {
		t.Errorf("initJWTSecret() secret = %v, want %v", secret, existingSecret)
	}
}

func TestInitJWTSecret_GenerateNew(t *testing.T) {
	st := store.NewMemoryStore()

	// Call initJWTSecret without config secret and empty storage
	secret, err := initJWTSecret(st, "")
	if err != nil {
		t.Fatalf("initJWTSecret() error = %v, want nil", err)
	}

	// Verify a secret was generated (should be non-empty)
	if secret == "" {
		t.Error("initJWTSecret() generated empty secret")
	}

	// Verify it was saved to storage
	storedSecret, err := st.GetConfig(jwtSecretConfigKey)
	if err != nil {
		t.Fatalf("GetConfig() error = %v, want nil", err)
	}
	if storedSecret != secret {
		t.Errorf("stored secret = %v, want %v", storedSecret, secret)
	}

	// Verify generated secret has reasonable length (base64 encoded 32 bytes = 44 chars)
	if len(secret) < 32 {
		t.Errorf("generated secret too short: len = %d, want >= 32", len(secret))
	}
}

func TestInitJWTSecret_ConfigOverridesStorage(t *testing.T) {
	st := store.NewMemoryStore()
	existingSecret := "existing-secret-in-storage"
	configSecret := "new-config-secret"

	// Pre-set a secret in storage
	err := st.SetConfig(jwtSecretConfigKey, existingSecret)
	if err != nil {
		t.Fatalf("SetConfig() error = %v, want nil", err)
	}

	// Call initJWTSecret with config secret (should override storage)
	secret, err := initJWTSecret(st, configSecret)
	if err != nil {
		t.Fatalf("initJWTSecret() error = %v, want nil", err)
	}

	if secret != configSecret {
		t.Errorf("initJWTSecret() secret = %v, want %v", secret, configSecret)
	}

	// Verify storage was updated with new config secret
	storedSecret, err := st.GetConfig(jwtSecretConfigKey)
	if err != nil {
		t.Fatalf("GetConfig() error = %v, want nil", err)
	}
	if storedSecret != configSecret {
		t.Errorf("stored secret = %v, want %v", storedSecret, configSecret)
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	// Generate multiple secrets and verify they're unique
	secrets := make(map[string]bool)
	for i := 0; i < 10; i++ {
		secret, err := generateRandomSecret(32)
		if err != nil {
			t.Fatalf("generateRandomSecret() error = %v, want nil", err)
		}

		// Check length (base64 encoding of 32 bytes)
		if len(secret) < 32 {
			t.Errorf("generateRandomSecret() len = %d, want >= 32", len(secret))
		}

		// Check uniqueness
		if secrets[secret] {
			t.Error("generateRandomSecret() generated duplicate secret")
		}
		secrets[secret] = true
	}
}

func TestInitJWTSecret_Persistence(t *testing.T) {
	st := store.NewMemoryStore()

	// First call - should generate new secret
	secret1, err := initJWTSecret(st, "")
	if err != nil {
		t.Fatalf("initJWTSecret() first call error = %v, want nil", err)
	}

	// Second call - should return same secret from storage
	secret2, err := initJWTSecret(st, "")
	if err != nil {
		t.Fatalf("initJWTSecret() second call error = %v, want nil", err)
	}

	if secret1 != secret2 {
		t.Errorf("initJWTSecret() not persistent: first = %v, second = %v", secret1, secret2)
	}
}
