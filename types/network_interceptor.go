package types

import (
	"github.com/go-rod/rod/lib/proto"
)

// networkInterceptor owns the state for CDP request interception (Fetch domain).
// All methods assume the caller holds Context.interceptLock.
type networkInterceptor struct {
	rules   []InterceptRule
	enabled bool
	// cancel cancels the EachEvent goroutine started by interceptEnable.
	cancel func()
}

// newNetworkInterceptor creates a networkInterceptor with no rules and interception disabled.
func newNetworkInterceptor() *networkInterceptor {
	return &networkInterceptor{}
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
// Must be called with interceptLock held.
func (ni *networkInterceptor) Enabled() bool {
	return ni.enabled
}

// SetEnabled sets whether interception is enabled.
// When disabling, any existing intercept listener goroutine is cancelled and rules are cleared.
// Must be called with interceptLock held.
func (ni *networkInterceptor) SetEnabled(enabled bool) {
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
// Must be called with interceptLock held.
func (ni *networkInterceptor) SetCancel(cancel func()) {
	if ni.cancel != nil {
		ni.cancel()
	}
	ni.cancel = cancel
}

// Cancel cancels the intercept listener goroutine if one is running, and
// clears the cancel function. Must be called with interceptLock held.
func (ni *networkInterceptor) Cancel() {
	if ni.cancel != nil {
		ni.cancel()
		ni.cancel = nil
	}
}

// AddRule appends an interception rule.
// Must be called with interceptLock held.
func (ni *networkInterceptor) AddRule(rule InterceptRule) {
	ni.rules = append(ni.rules, rule)
}

// Rules returns a copy of the current interception rules.
// Must be called with interceptLock held.
func (ni *networkInterceptor) Rules() []InterceptRule {
	rules := make([]InterceptRule, len(ni.rules))
	copy(rules, ni.rules)
	return rules
}
