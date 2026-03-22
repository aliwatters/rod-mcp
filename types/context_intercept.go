package types

import (
	"github.com/go-rod/rod/lib/proto"
)

// InterceptRule defines a rule for how to handle an intercepted request.
type InterceptRule struct {
	URLPattern  string
	Action      string // "mock", "block", "fail"
	Status      int
	Headers     []*proto.FetchHeaderEntry
	Body        []byte
	ErrorReason proto.NetworkErrorReason
}

// InterceptEnabled returns whether request interception is currently enabled.
func (ctx *Context) InterceptEnabled() bool {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	return ctx.interceptEnabled
}

// SetInterceptEnabled sets whether interception is enabled.
// When disabling, any existing intercept listener goroutine is cancelled.
func (ctx *Context) SetInterceptEnabled(enabled bool) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.interceptEnabled = enabled
	if !enabled {
		ctx.interceptRules = nil
		if ctx.interceptCancel != nil {
			ctx.interceptCancel()
			ctx.interceptCancel = nil
		}
	}
}

// SetInterceptCancel stores the cancel function for the intercept EachEvent goroutine.
func (ctx *Context) SetInterceptCancel(cancel func()) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	// Cancel any prior listener before replacing.
	if ctx.interceptCancel != nil {
		ctx.interceptCancel()
	}
	ctx.interceptCancel = cancel
}

// AddInterceptRule appends an interception rule.
func (ctx *Context) AddInterceptRule(rule InterceptRule) {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	ctx.interceptRules = append(ctx.interceptRules, rule)
}

// InterceptRules returns a copy of the current interception rules.
func (ctx *Context) InterceptRules() []InterceptRule {
	ctx.stateLock.Lock()
	defer ctx.stateLock.Unlock()
	rules := make([]InterceptRule, len(ctx.interceptRules))
	copy(rules, ctx.interceptRules)
	return rules
}
