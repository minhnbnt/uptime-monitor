package domain

type Scope string

const (
	ScopeApp  Scope = "app"
	ScopePing Scope = "ping"
)

func DefaultScopes() []string {
	return []string{string(ScopeApp)}
}
