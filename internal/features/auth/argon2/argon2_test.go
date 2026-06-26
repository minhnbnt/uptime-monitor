package argon2

import (
	"encoding/base64"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

func testEncoder() *Argon2PasswordEncoder {
	return &Argon2PasswordEncoder{
		config: config.NewArgon2Config(64*1024, 2, 1, 16, 32),
	}
}

func TestVerify_ValidPassword(t *testing.T) {
	encoder := testEncoder()
	hashed, err := encoder.Encode("correct-password")
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	match, err := encoder.Verify("correct-password", hashed)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !match {
		t.Error("expected match for correct password")
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	encoder := testEncoder()
	hashed, err := encoder.Encode("real-password")
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	match, err := encoder.Verify("wrong-password", hashed)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if match {
		t.Error("expected no match for wrong password")
	}
}

func TestVerify_InvalidHash(t *testing.T) {
	encoder := testEncoder()
	_, err := encoder.Verify("password", "not-a-valid-argon2-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash")
	}
}

func TestVerify_IncompatibleVersion(t *testing.T) {
	encoder := testEncoder()
	salt := base64.RawStdEncoding.EncodeToString([]byte("test-salt-12345678"))
	hash := base64.RawStdEncoding.EncodeToString([]byte("test-hash-value-123456789012"))
	badHash := "$argon2id$v=99$m=65536,t=2,p=1$" + salt + "$" + hash
	_, err := encoder.Verify("password", badHash)
	if err != ErrIncompatibleVersion {
		t.Errorf("got %v, want ErrIncompatibleVersion", err)
	}
}

func TestVerify_EmptyHash(t *testing.T) {
	encoder := testEncoder()
	_, err := encoder.Verify("password", "")
	if err == nil {
		t.Fatal("expected error for empty hash")
	}
}

func TestDecodeArgon2Hash(t *testing.T) {
	salt := base64.RawStdEncoding.EncodeToString([]byte("test-salt-123456"))
	hash := base64.RawStdEncoding.EncodeToString([]byte("test-hash-value-1234567890"))
	validHash := "$argon2id$v=19$m=16384,t=2,p=1$" + salt + "$" + hash

	t.Run("valid hash", func(t *testing.T) {
		p, gotSalt, gotHash, err := decodeArgon2Hash(validHash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.memory != 16384 {
			t.Errorf("memory = %d, want 16384", p.memory)
		}
		if p.iterations != 2 {
			t.Errorf("iterations = %d, want 2", p.iterations)
		}
		if p.parallelism != 1 {
			t.Errorf("parallelism = %d, want 1", p.parallelism)
		}
		if string(gotSalt) != "test-salt-123456" {
			t.Errorf("salt = %q", gotSalt)
		}
		if string(gotHash) != "test-hash-value-1234567890" {
			t.Errorf("hash = %q", gotHash)
		}
	})

	t.Run("wrong segment count", func(t *testing.T) {
		_, _, _, err := decodeArgon2Hash("$argon2id$v=19$m=1,t=1,p=1$salt")
		if err != ErrInvalidHash {
			t.Errorf("got %v, want ErrInvalidHash", err)
		}
	})

	t.Run("incompatible version", func(t *testing.T) {
		h := "$argon2id$v=99$m=1,t=1,p=1$c2FsdA$aGFzaA"
		_, _, _, err := decodeArgon2Hash(h)
		if err != ErrIncompatibleVersion {
			t.Errorf("got %v, want ErrIncompatibleVersion", err)
		}
	})

	t.Run("invalid salt base64", func(t *testing.T) {
		h := "$argon2id$v=19$m=1,t=1,p=1$!!!$aGFzaA"
		_, _, _, err := decodeArgon2Hash(h)
		if err == nil {
			t.Fatal("expected error for invalid base64 salt")
		}
	})

	t.Run("invalid hash base64", func(t *testing.T) {
		s := base64.RawStdEncoding.EncodeToString([]byte("salt1234"))
		h := "$argon2id$v=19$m=1,t=1,p=1$" + s + "$!!!"
		_, _, _, err := decodeArgon2Hash(h)
		if err == nil {
			t.Fatal("expected error for invalid base64 hash")
		}
	})

	t.Run("different argon2 parameters", func(t *testing.T) {
		s := base64.RawStdEncoding.EncodeToString([]byte("salt123456789012"))
		h := base64.RawStdEncoding.EncodeToString([]byte("hash1234567890123456"))
		encoded := "$argon2id$v=19$m=32768,t=4,p=2$" + s + "$" + h
		p, _, _, err := decodeArgon2Hash(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.memory != 32768 || p.iterations != 4 || p.parallelism != 2 {
			t.Errorf("got m=%d,t=%d,p=%d", p.memory, p.iterations, p.parallelism)
		}
	})
}
