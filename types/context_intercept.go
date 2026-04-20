package types

// InterceptEnabled returns whether request interception is currently enabled.
func (ctx *Context) InterceptEnabled() bool {
	ctx.interceptLock.Lock()
	defer ctx.interceptLock.Unlock()
	return ctx.intercept.Enabled()
}

// SetInterceptEnabled sets whether interception is enabled.
// When disabling, any existing intercept listener goroutine is cancelled.
func (ctx *Context) SetInterceptEnabled(enabled bool) {
	ctx.interceptLock.Lock()
	defer ctx.interceptLock.Unlock()
	ctx.intercept.SetEnabled(enabled)
}

// SetInterceptCancel stores the cancel function for the intercept EachEvent goroutine.
func (ctx *Context) SetInterceptCancel(cancel func()) {
	ctx.interceptLock.Lock()
	defer ctx.interceptLock.Unlock()
	ctx.intercept.SetCancel(cancel)
}

// AddInterceptRule appends an interception rule.
func (ctx *Context) AddInterceptRule(rule InterceptRule) {
	ctx.interceptLock.Lock()
	defer ctx.interceptLock.Unlock()
	ctx.intercept.AddRule(rule)
}

// InterceptRules returns a copy of the current interception rules.
func (ctx *Context) InterceptRules() []InterceptRule {
	ctx.interceptLock.Lock()
	defer ctx.interceptLock.Unlock()
	return ctx.intercept.Rules()
}
