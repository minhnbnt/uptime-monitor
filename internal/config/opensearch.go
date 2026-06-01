package config

import (
	"os"
	"strings"

	"github.com/opensearch-project/opensearch-go"
	"github.com/samber/do/v2"
)

func newOpensearchConfig(i do.Injector) (*opensearch.Config, error) {

	opensearchAddrsEnv := os.Getenv("OPENSEARCH_ADDRESSES")
	addrs := strings.Split(opensearchAddrsEnv, ",")

	config := &opensearch.Config{}
	if len(addrs) > 0 {
		config.Addresses = addrs
	}

	return config, nil
}

func newOpensearchClient(i do.Injector) (*opensearch.Client, error) {
	config := do.MustInvoke[*opensearch.Config](i)
	return opensearch.NewClient(*config)
}

func RegisterOpensearchClient(i do.Injector) {
	do.Provide(i, newOpensearchConfig)
	do.Provide(i, newOpensearchClient)
}
