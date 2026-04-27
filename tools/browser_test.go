package tools

import "testing"

// TestWrapFunctionExpression exercises the script-wrapping logic used by
// rod_evaluate. Without wrapping, function expressions evaluate to a function
// reference and serialize as "{}" via ReturnByValue — see issue #280.
func TestWrapFunctionExpression(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "arrow function in parens",
			in:   "() => document.title",
			want: "(() => document.title)()",
		},
		{
			name: "anonymous function expression",
			in:   "function () { return 1 }",
			want: "(function () { return 1 })()",
		},
		{
			name: "async arrow function (issue #280)",
			in:   "async () => { return { ok: true } }",
			want: "(async () => { return { ok: true } })()",
		},
		{
			name: "async function expression",
			in:   "async function () { return 1 }",
			want: "(async function () { return 1 })()",
		},
		{
			name: "leading whitespace before async arrow",
			in:   "  async () => 1",
			want: "(  async () => 1)()",
		},
		{
			name: "bare expression is left alone",
			in:   "document.title",
			want: "document.title",
		},
		{
			name: "identifier that starts with async is not wrapped",
			in:   "asyncify()",
			want: "asyncify()",
		},
		{
			name: "identifier that starts with function is not wrapped",
			in:   "functionFoo()",
			want: "functionFoo()",
		},
		{
			name: "empty script is left alone",
			in:   "",
			want: "",
		},
		{
			name: "already-invoked IIFE is left alone",
			in:   "(() => 1)()",
			want: "(() => 1)()",
		},
		{
			name: "already-invoked IIFE with arguments is left alone",
			in:   "(function () { return 1 })()",
			want: "(function () { return 1 })()",
		},
		{
			name: "parenthesised arrow that is not invoked is wrapped",
			in:   "(() => 1)",
			want: "((() => 1))()",
		},
		{
			name: "async assignment is not wrapped",
			in:   "async = 1",
			want: "async = 1",
		},
		{
			name: "async followed by non-function token is not wrapped",
			in:   "async + 1",
			want: "async + 1",
		},
		{
			name: "async with extra whitespace before arrow is wrapped",
			in:   "async   () => 1",
			want: "(async   () => 1)()",
		},
		{
			name: "async with newline before function keyword is wrapped",
			in:   "async\nfunction () { return 1 }",
			want: "(async\nfunction () { return 1 })()",
		},
		{
			name: "asyncFoo identifier is not wrapped",
			in:   "asyncFoo()",
			want: "asyncFoo()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapFunctionExpression(tt.in)
			if got != tt.want {
				t.Errorf("wrapFunctionExpression(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
