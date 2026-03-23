// performance_vitals.js — collects Core Web Vitals via the Performance API.
// Returns a JSON-encoded object with timing metrics.
() => {
	const result = {};
	const entries = performance.getEntriesByType('navigation');
	if (entries.length > 0) {
		const nav = entries[0];
		result.ttfb = Math.round(nav.responseStart - nav.requestStart);
		result.domContentLoaded = Math.round(nav.domContentLoadedEventEnd - nav.startTime);
		result.load = Math.round(nav.loadEventEnd - nav.startTime);
		result.domInteractive = Math.round(nav.domInteractive - nav.startTime);
	}
	const paintEntries = performance.getEntriesByType('paint');
	for (const entry of paintEntries) {
		if (entry.name === 'first-paint') result.fp = Math.round(entry.startTime);
		if (entry.name === 'first-contentful-paint') result.fcp = Math.round(entry.startTime);
	}
	const lcpEntries = performance.getEntriesByType('largest-contentful-paint');
	if (lcpEntries.length > 0) {
		result.lcp = Math.round(lcpEntries[lcpEntries.length - 1].startTime);
	}
	const layoutShifts = performance.getEntriesByType('layout-shift');
	let cls = 0;
	for (const entry of layoutShifts) {
		if (!entry.hadRecentInput) cls += entry.value;
	}
	result.cls = Math.round(cls * 1000) / 1000;
	return JSON.stringify(result);
}
