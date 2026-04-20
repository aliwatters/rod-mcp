package types

import (
	"errors"
	"fmt"

	"github.com/go-rod/rod"
)

// TabInfo represents a tab's metadata for listing.
type TabInfo struct {
	Index    int    `json:"index"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	IsActive bool   `json:"is_active"`
}

// NewTab creates a new tab, optionally navigating to a URL, and makes it the active page.
func (ctx *Context) NewTab(url string) (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()

	var page *rod.Page
	var err error
	if url != "" {
		page, err = ctx.createPage(url)
	} else {
		page, err = ctx.createPage()
	}
	if err != nil {
		return nil, err
	}
	ctx.page = page
	return page, nil
}

// ListTabs returns info about all open tabs, marking which is active.
func (ctx *Context) ListTabs() ([]TabInfo, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("list tabs: %w", err)
	}

	tabs := make([]TabInfo, 0, len(pages))
	for i, p := range pages {
		info, err := p.Info()
		if err != nil {
			return nil, fmt.Errorf("get tab info: %w", err)
		}
		tabs = append(tabs, TabInfo{
			Index:    i,
			Title:    info.Title,
			URL:      info.URL,
			IsActive: ctx.page != nil && p.TargetID == ctx.page.TargetID,
		})
	}
	return tabs, nil
}

// SelectTab switches the active page to the tab at the given index.
func (ctx *Context) SelectTab(index int) (*rod.Page, error) {
	if err := ctx.initial(); err != nil {
		return nil, err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return nil, fmt.Errorf("list tabs: %w", err)
	}
	if index < 0 || index >= len(pages) {
		return nil, fmt.Errorf("tab index %d out of range (0-%d)", index, len(pages)-1)
	}
	ctx.page = pages[index]
	return ctx.page, nil
}

// CloseTab closes the tab at the given index. If it's the active tab,
// switches to the nearest remaining tab.
func (ctx *Context) CloseTab(index int) error {
	if err := ctx.initial(); err != nil {
		return err
	}
	ctx.browserLock.Lock()
	defer ctx.browserLock.Unlock()

	pages, err := ctx.browser.Pages()
	if err != nil {
		return fmt.Errorf("list tabs: %w", err)
	}
	if index < 0 || index >= len(pages) {
		return fmt.Errorf("tab index %d out of range (0-%d)", index, len(pages)-1)
	}
	if len(pages) == 1 {
		return errors.New("cannot close the last tab")
	}

	target := pages[index]
	isActive := ctx.page != nil && target.TargetID == ctx.page.TargetID

	if err := target.Close(); err != nil {
		return fmt.Errorf("close tab: %w", err)
	}

	if isActive {
		// Switch to nearest tab
		remaining, err := ctx.browser.Pages()
		if err != nil {
			return fmt.Errorf("list tabs after close: %w", err)
		}
		if len(remaining) > 0 {
			newIndex := index
			if newIndex >= len(remaining) {
				newIndex = len(remaining) - 1
			}
			ctx.page = remaining[newIndex]
		} else {
			ctx.page = nil
		}
	}

	return nil
}
