// wait_detached.js — resolves when the element matching %SELECTOR% is removed from the DOM.
// The placeholder %SELECTOR% is replaced at runtime with a JSON-encoded CSS selector string.
() => new Promise((resolve) => {
	const check = () => {
		if (!document.querySelector(%SELECTOR%)) { resolve(true); return; }
		requestAnimationFrame(check);
	};
	check();
})
