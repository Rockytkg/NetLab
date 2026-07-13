package rbac

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const modelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act)
`

var defaultPolicies = [][]string{
	{"super_admin", "/api/admin/*", "(GET|POST|PUT|DELETE|PATCH)"},
	{"super_admin", "/api/auth/*", "(GET|POST|PUT|DELETE|PATCH)"},
	{"admin", "/api/admin/*", "(GET|POST|PUT|DELETE|PATCH)"},
	{"admin", "/api/auth/*", "(GET|POST|PUT|DELETE|PATCH)"},
	{"editor", "/api/auth/*", "(GET|POST|PUT|DELETE|PATCH)"},
	{"viewer", "/api/auth/*", "(GET|POST|PUT|DELETE|PATCH)"},
}

// NewEnforcer creates the RBAC enforcer backed by the project GORM database.
func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("load rbac model: %w", err)
	}

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}
	if err := seedPolicies(enforcer); err != nil {
		return nil, err
	}
	return enforcer, nil
}

func seedPolicies(enforcer *casbin.Enforcer) error {
	for _, p := range defaultPolicies {
		ok, err := enforcer.HasPolicy(p)
		if err != nil {
			return fmt.Errorf("check rbac policy %v: %w", p, err)
		}
		if ok {
			continue
		}
		if _, err := enforcer.AddPolicy(p); err != nil {
			return fmt.Errorf("seed rbac policy %v: %w", p, err)
		}
	}
	return nil
}
