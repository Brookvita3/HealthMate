package common

type ContextKey string

const (
	UserIdKey ContextKey = "userId"
	EmailKey  ContextKey = "email"
	Role      ContextKey = "role"
)
