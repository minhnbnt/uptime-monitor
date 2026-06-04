package config

import "github.com/samber/do/v2"

type Argon2Config struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

func RegisterArgon2Config(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Argon2Config, error) {
		return &Argon2Config{
			memory:      1 << 14,
			iterations:  2,
			parallelism: 1,
			saltLength:  16,
			keyLength:   32,
		}, nil
	})
}

func (a2 *Argon2Config) GetMemory() uint32 {
	return a2.memory
}

func (a2 *Argon2Config) GetIterations() uint32 {
	return a2.iterations
}

func (a2 *Argon2Config) GetParallelism() uint8 {
	return a2.parallelism
}

func (a2 *Argon2Config) GetSaltLength() uint32 {
	return a2.saltLength
}

func (a2 *Argon2Config) GetKeyLength() uint32 {
	return a2.keyLength
}
