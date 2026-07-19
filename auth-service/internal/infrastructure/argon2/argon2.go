package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/do/v2"
	"golang.org/x/crypto/argon2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
)

type PasswordEncoder struct {
	config *config.Argon2Config
}

func RegisterPasswordEncoder(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PasswordEncoder, error) {
		config := do.MustInvoke[*config.Argon2Config](i)
		return &PasswordEncoder{config: config}, nil
	})
}

func (e *PasswordEncoder) Encode(password string) (string, error) {

	salt := make([]byte, e.config.GetSaltLength())
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	encoded := argon2.IDKey(
		[]byte(password), salt,
		e.config.GetIterations(),
		e.config.GetMemory(),
		e.config.GetParallelism(),
		e.config.GetKeyLength(),
	)

	encoder := base64.RawStdEncoding

	b64Salt := encoder.EncodeToString(salt)
	b64Hash := encoder.EncodeToString(encoded)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, e.config.GetMemory(),
		e.config.GetIterations(),
		e.config.GetParallelism(),
		b64Salt, b64Hash,
	)

	return encodedHash, nil
}

func (e *PasswordEncoder) Verify(password, encodedHash string) (bool, error) {

	p, salt, hash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false, err
	}

	rawHash := argon2.IDKey(
		[]byte(password), salt,
		p.iterations,
		p.memory,
		p.parallelism,
		e.config.GetKeyLength(),
	)

	compareResult := subtle.ConstantTimeCompare(rawHash, hash)
	return compareResult == 1, nil
}

var (
	ErrInvalidHash         = errors.New("invalid hash")
	ErrIncompatibleVersion = errors.New("incompatible version")
)

func decodeArgon2Hash(encodedHash string) (p *params, salt, hash []byte, err error) {

	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}

	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p = &params{}
	_, err = fmt.Sscanf(
		vals[3], "m=%d,t=%d,p=%d",
		&p.memory, &p.iterations, &p.parallelism,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	decoder := base64.RawStdEncoding.Strict()

	salt, err = decoder.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}

	p.saltLength = uint32(len(salt))

	hash, err = decoder.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}

	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}

type params struct {
	saltLength  uint32
	keyLength   uint32
	iterations  uint32
	memory      uint32
	parallelism uint8
}
