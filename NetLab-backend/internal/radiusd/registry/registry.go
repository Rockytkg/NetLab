// Package registry is the plugin pipeline registry for the RADIUS server. It
// holds the cross-cutting auth/accounting/EAP plugins that run for every
// request: password validators, policy checkers, response enhancers, auth
// guards, accounting handlers, and EAP handlers.
//
// It is distinct from the internal/radiusd/vendors package, which is the
// registry of vendor definitions (vendor codes plus their attribute parsers and
// response builders). Vendor parser/builder registration and lookup belong in
// vendors, not here; this package intentionally does not deal with vendors.
//
// 本包移植自 github.com/talkincode/toughradius（MIT License）。
package registry

import (
	"sort"
	"sync"

	"netlab-backend/internal/radiusd/plugins/accounting"
	"netlab-backend/internal/radiusd/plugins/auth"
	"netlab-backend/internal/radiusd/plugins/eap"
)

// Registry holds plugin registrations
type Registry struct {
	passwordValidators map[string]auth.PasswordValidator
	policyCheckers     []auth.PolicyChecker
	responseEnhancers  []auth.ResponseEnhancer
	authGuards         []auth.Guard
	acctHandlers       []accounting.AccountingHandler
	eapHandlers        map[uint8]eap.EAPHandler // EAP handlers indexed by EAP type
	mu                 sync.RWMutex
}

var globalRegistry = newRegistry()

func newRegistry() *Registry {
	return &Registry{
		passwordValidators: make(map[string]auth.PasswordValidator),
		policyCheckers:     make([]auth.PolicyChecker, 0),
		responseEnhancers:  make([]auth.ResponseEnhancer, 0),
		authGuards:         make([]auth.Guard, 0),
		acctHandlers:       make([]accounting.AccountingHandler, 0),
		eapHandlers:        make(map[uint8]eap.EAPHandler),
	}
}

// RegisterPasswordValidator registers a password validator
func RegisterPasswordValidator(validator auth.PasswordValidator) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.passwordValidators[validator.Name()] = validator
}

// GetPasswordValidators returns all password validators
func GetPasswordValidators() []auth.PasswordValidator {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	validators := make([]auth.PasswordValidator, 0, len(globalRegistry.passwordValidators))
	for _, v := range globalRegistry.passwordValidators {
		validators = append(validators, v)
	}
	return validators
}

// GetPasswordValidator returns a password validator by name
func GetPasswordValidator(name string) (auth.PasswordValidator, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	v, ok := globalRegistry.passwordValidators[name]
	return v, ok
}

// RegisterPolicyChecker registers a profile checker. Re-registering a checker
// with the same name replaces the previous entry (idempotent, safe for
// in-process server rebuilds).
func RegisterPolicyChecker(checker auth.PolicyChecker) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	for i, c := range globalRegistry.policyCheckers {
		if c.Name() == checker.Name() {
			globalRegistry.policyCheckers[i] = checker
			sort.Slice(globalRegistry.policyCheckers, func(i, j int) bool {
				return globalRegistry.policyCheckers[i].Order() < globalRegistry.policyCheckers[j].Order()
			})
			return
		}
	}
	globalRegistry.policyCheckers = append(globalRegistry.policyCheckers, checker)
	// Sort by order
	sort.Slice(globalRegistry.policyCheckers, func(i, j int) bool {
		return globalRegistry.policyCheckers[i].Order() < globalRegistry.policyCheckers[j].Order()
	})
}

// GetPolicyCheckers returns all profile checkers (sorted by order)
func GetPolicyCheckers() []auth.PolicyChecker {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	// Returns a copy
	checkers := make([]auth.PolicyChecker, len(globalRegistry.policyCheckers))
	copy(checkers, globalRegistry.policyCheckers)
	return checkers
}

// RegisterResponseEnhancer registers a response enhancer (idempotent by name).
func RegisterResponseEnhancer(enhancer auth.ResponseEnhancer) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	for i, e := range globalRegistry.responseEnhancers {
		if e.Name() == enhancer.Name() {
			globalRegistry.responseEnhancers[i] = enhancer
			return
		}
	}
	globalRegistry.responseEnhancers = append(globalRegistry.responseEnhancers, enhancer)
}

// GetResponseEnhancers returns all response enhancers
func GetResponseEnhancers() []auth.ResponseEnhancer {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	enhancers := make([]auth.ResponseEnhancer, len(globalRegistry.responseEnhancers))
	copy(enhancers, globalRegistry.responseEnhancers)
	return enhancers
}

// RegisterAuthGuard registers an authentication guard (idempotent by name).
func RegisterAuthGuard(guard auth.Guard) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	for i, g := range globalRegistry.authGuards {
		if g.Name() == guard.Name() {
			globalRegistry.authGuards[i] = guard
			return
		}
	}
	globalRegistry.authGuards = append(globalRegistry.authGuards, guard)
}

// GetAuthGuards returns all authentication guards
func GetAuthGuards() []auth.Guard {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	guards := make([]auth.Guard, len(globalRegistry.authGuards))
	copy(guards, globalRegistry.authGuards)
	return guards
}

// RegisterAccountingHandler registers an accounting handler (idempotent by name).
func RegisterAccountingHandler(handler accounting.AccountingHandler) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	for i, h := range globalRegistry.acctHandlers {
		if h.Name() == handler.Name() {
			globalRegistry.acctHandlers[i] = handler
			return
		}
	}
	globalRegistry.acctHandlers = append(globalRegistry.acctHandlers, handler)
}

// GetAccountingHandlers returns all accounting handlers
func GetAccountingHandlers() []accounting.AccountingHandler {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	handlers := make([]accounting.AccountingHandler, len(globalRegistry.acctHandlers))
	copy(handlers, globalRegistry.acctHandlers)
	return handlers
}

// RegisterEAPHandler registers an EAP handler
func RegisterEAPHandler(handler eap.EAPHandler) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.eapHandlers[handler.EAPType()] = handler
}

// ResetEAPHandlers clears all registered EAP handlers. Called before the
// conditional EAP registration during an in-process server rebuild so that a
// previously enabled method does not leak into a new configuration.
func ResetEAPHandlers() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.eapHandlers = make(map[uint8]eap.EAPHandler)
}

// GetEAPHandler returns the handler for a given EAP type
func GetEAPHandler(eapType uint8) (eap.EAPHandler, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	handler, ok := globalRegistry.eapHandlers[eapType]
	return handler, ok
}

// GetAllEAPHandlers returns all EAP handlers
func GetAllEAPHandlers() map[uint8]eap.EAPHandler {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	handlers := make(map[uint8]eap.EAPHandler, len(globalRegistry.eapHandlers))
	for k, v := range globalRegistry.eapHandlers {
		handlers[k] = v
	}
	return handlers
}

// GetHandler implements the eap.HandlerRegistry interface
func (r *Registry) GetHandler(eapType uint8) (eap.EAPHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.eapHandlers[eapType]
	return handler, ok
}

// GetGlobalRegistry returns the global registry instance
// Used to implement the eap.HandlerRegistry interface
func GetGlobalRegistry() *Registry {
	return globalRegistry
}

// ResetForTest clears the registry. Test helper only.
func ResetForTest() {
	globalRegistry = newRegistry()
}
