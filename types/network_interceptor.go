package types

import (
	"github.com/go-rod/rod/lib/proto"
)

// NetworkInterceptor owns the state for CDP request interception (Fetch domain).
// All methods assume the caller holds Context.stateLock.
type NetworkInterceptor struct {
	rules   []InterceptRule
	enabled bool
	// cancel cancels the EachEvent goroutine started by interceptEnable.
	cancel func()
}

// NewNetworkInterceptor creates a NetworkInterceptor with no rules and interception disabled.
func NewNetworkInterceptor() *NetworkInterceptor {
	return &NetworkInterceptor{}
}

// InterceptRule defines a rule for how to handle an intercepted request.
type InterceptRule struct {
	URLPattern  string
	Action      string // "mock", "block", "fail"
	Status      int
	Headers     []*proto.FetchHeaderEntry
	Body        []byte
	ErrorReason proto.NetworkErrorReason
}

// Enabled returns whether request interception is currently enabled.
// Must be called with stateLock held.
func (ni *NetworkInterceptor) Enabled() bool {
	return ni.enabled
}

// SetEnabled sets whether interception is enabled.
// When disabling, any existing intercept listener goroutine is cancelled and rules are cleared.
// Must be called with stateLock held.
func (ni *NetworkInterceptor) SetEnabled(enabled bool) {
	ni.enabled = enabled
	if !enabled {
		ni.rules = nil
		if ni.cancel != nil {
			ni.cancel()
			ni.cancel = nil
		}
	}
}

// SetCancel stores the cancel function for the intercept EachEvent goroutine.
// Any prior listener is cancelled before replacing.
// Must be called with stateLock held.
func (ni *NetworkInterceptor) SetCancel(cancel func()) {
	if ni.cancel != nil {
		ni.cancel()
	}
	ni.cancel = cancel
}

// Cancel cancels the intercept listener goroutine if one is running, and
// clears the cancel function. Must be called with stateLock held.
func (ni *NetworkInterceptor) Cancel() {
	if ni.cancel != nil {
		ni.cancel()
		ni.cancel = nil
	}
}

// AddRule appends an interception rule.
// Must be called with stateLock held.
func (ni *NetworkInterceptor) AddRule(rule InterceptRule) {
	ni.rules = append(ni.rules, rule)
}

// Rules returns a copy of the current interception rules.
// Must be called with stateLock held.
func (ni *NetworkInterceptor) Rules() []InterceptRule {
	rules := make([]InterceptRule, len(ni.rules))
	copy(rules, ni.rules)
	return rules
}
