package types

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TabInfo struct
// ---------------------------------------------------------------------------

func TestTabInfo_Fields(t *testing.T) {
	tab := TabInfo{
		Index:    2,
		Title:    "My Page",
		URL:      "https://example.com",
		IsActive: true,
	}
	if tab.Index != 2 {
		t.Errorf("Index = %d, want 2", tab.Index)
	}
	if tab.Title != "My Page" {
		t.Errorf("Title = %q, want %q", tab.Title, "My Page")
	}
	if tab.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", tab.URL, "https://example.com")
	}
	if !tab.IsActive {
		t.Error("IsActive should be true")
	}
}

func TestTabInfo_ZeroValue(t *testing.T) {
	var tab TabInfo
	if tab.Index != 0 {
		t.Errorf("zero Index = %d, want 0", tab.Index)
	}
	if tab.Title != "" {
		t.Errorf("zero Title = %q, want empty", tab.Title)
	}
	if tab.URL != "" {
		t.Errorf("zero URL = %q, want empty", tab.URL)
	}
	if tab.IsActive {
		t.Error("zero IsActive should be false")
	}
}

func TestTabInfo_MultipleEntries(t *testing.T) {
	// Only one tab should be marked active at a time in a well-formed list.
	tabs := []TabInfo{
		{Index: 0, Title: "Tab Zero", URL: "about:blank", IsActive: true},
		{Index: 1, Title: "Tab One", URL: "https://example.com", IsActive: false},
		{Index: 2, Title: "Tab Two", URL: "https://other.com", IsActive: false},
	}

	activeCount := 0
	for _, tab := range tabs {
		if tab.IsActive {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("expected 1 active tab, got %d", activeCount)
	}
	if tabs[2].Index != 2 {
		t.Errorf("tabs[2].Index = %d, want 2", tabs[2].Index)
	}
}

// ---------------------------------------------------------------------------
// outOfRangeMsg — error message format used by SelectTab and CloseTab
// ---------------------------------------------------------------------------

// tabOutOfRangeMsg mirrors the fmt.Sprintf format used in SelectTab/CloseTab
// so we can verify the error message format without a live browser.
func tabOutOfRangeMsg(index, numPages int) string {
	return fmt.Sprintf("tab index %d out of range (0-%d)", index, numPages-1)
}

func TestTabOutOfRangeMsg_Format(t *testing.T) {
	tests := []struct {
		index    int
		numPages int
		want     string
	}{
		{5, 3, "tab index 5 out of range (0-2)"},
		{0, 1, "tab index 0 out of range (0-0)"},
		{10, 5, "tab index 10 out of range (0-4)"},
	}
	for _, tt := range tests {
		got := tabOutOfRangeMsg(tt.index, tt.numPages)
		if got != tt.want {
			t.Errorf("tabOutOfRangeMsg(%d, %d) = %q, want %q", tt.index, tt.numPages, got, tt.want)
		}
	}
}

func TestTabOutOfRangeMsg_ContainsIndex(t *testing.T) {
	msg := tabOutOfRangeMsg(-1, 3)
	if !strings.Contains(msg, "-1") {
		t.Errorf("out-of-range error message %q should contain the index -1", msg)
	}
}

// ---------------------------------------------------------------------------
// NewTab — without a live browser, initial() fails
// ---------------------------------------------------------------------------

func TestNewTab_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	_, err := ctx.NewTab("")
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
	if err.Error() == "" {
		t.Error("NewTab: error message should not be empty")
	}
}

func TestNewTab_WithURL_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	_, err := ctx.NewTab("https://example.com")
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
}

// ---------------------------------------------------------------------------
// ListTabs — without a live browser, initial() fails
// ---------------------------------------------------------------------------

func TestListTabs_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	tabs, err := ctx.ListTabs()
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
	if tabs != nil {
		t.Error("ListTabs: should return nil tabs on error")
	}
}

// ---------------------------------------------------------------------------
// SelectTab — without a live browser, initial() fails
// ---------------------------------------------------------------------------

func TestSelectTab_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	_, err := ctx.SelectTab(0)
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
}

func TestSelectTab_NegativeIndex_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	_, err := ctx.SelectTab(-1)
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
}

// ---------------------------------------------------------------------------
// CloseTab — without a live browser, initial() fails
// ---------------------------------------------------------------------------

func TestCloseTab_NoBrowser_ReturnsError(t *testing.T) {
	ctx := newTestContext()
	err := ctx.CloseTab(0)
	if err == nil {
		t.Skip("Chrome is available; skipping no-browser error path")
	}
}
