package domain

type Scope string

const (
	ScopeAPI  Scope = "api"
	ScopePing Scope = "ping"
)

func DefaultScopes() []string {
	return []string{string(ScopeAPI), string(ScopePing)}
}
