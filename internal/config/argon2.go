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
		return &Argon2Config{
			config: cfg.Argon2,
		}, nil
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

type Argon2Cfg struct {
	Memory      uint32 `mapstructure:"memory"`
	Iterations  uint32 `mapstructure:"iterations"`
	Parallelism uint8  `mapstructure:"parallelism"`
	SaltLength  uint32 `mapstructure:"salt_length"`
	KeyLength   uint32 `mapstructure:"key_length"`
}
