package config

import (
	"github.com/samber/do/v2"
)

type Argon2Config struct {
	config Argon2Cfg
}

func RegisterArgon2Config(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Argon2Config, error) {
		cfg := do.MustInvoke[*Config](i)
		return &Argon2Config{config: cfg.Argon2}, nil
	})
}

func (a2 *Argon2Config) GetMemory() uint32 {
	return a2.config.Memory
}

func (a2 *Argon2Config) GetIterations() uint32 {
	return a2.config.Iterations
}

func (a2 *Argon2Config) GetParallelism() uint8 {
	return a2.config.Parallelism
}

func (a2 *Argon2Config) GetSaltLength() uint32 {
	return a2.config.SaltLength
}

func (a2 *Argon2Config) GetKeyLength() uint32 {
	return a2.config.KeyLength
}

func NewArgon2Config(memory, iterations uint32, parallelism uint8, saltLength, keyLength uint32) *Argon2Config {
	return &Argon2Config{
		config: Argon2Cfg{
			Memory:      memory,
			Iterations:  iterations,
			Parallelism: parallelism,
			SaltLength:  saltLength,
			KeyLength:   keyLength,
		},
	}
}
