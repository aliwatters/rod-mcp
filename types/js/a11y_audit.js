// a11y_audit.js — injected as a self-invoking function by buildJSAuditScript.
// The placeholder %ROOT_EXPR% is replaced at runtime with the root element expression.
function() {
	const root = %ROOT_EXPR%;
	const issues = [];

	// Rule: img-alt — images without alt text
	root.querySelectorAll('img').forEach(img => {
		if (!img.hasAttribute('alt')) {
			issues.push({
				severity: 'error',
				rule: 'img-alt',
				element: img.outerHTML.substring(0, 120),
				selector: cssPath(img),
				message: 'Image missing alt text'
			});
		}
	});

	// Rule: button-name — buttons without accessible names
	root.querySelectorAll('button, [role="button"]').forEach(btn => {
		const name = btn.textContent.trim() ||
			btn.getAttribute('aria-label') ||
			btn.getAttribute('aria-labelledby') ||
			btn.getAttribute('title');
		if (!name) {
			issues.push({
				severity: 'error',
				rule: 'button-name',
				element: btn.outerHTML.substring(0, 120),
				selector: cssPath(btn),
				message: 'Button has no accessible name (no text, aria-label, or aria-labelledby)'
			});
		}
	});

	// Rule: input-label — form inputs without labels
	root.querySelectorAll('input:not([type="hidden"]):not([type="submit"]):not([type="button"]):not([type="reset"]), select, textarea').forEach(input => {
		const id = input.id;
		const hasLabel = (id && root.querySelector('label[for="' + id + '"]')) ||
			input.closest('label') ||
			input.getAttribute('aria-label') ||
			input.getAttribute('aria-labelledby') ||
			input.getAttribute('title');
		if (!hasLabel) {
			issues.push({
				severity: 'error',
				rule: 'input-label',
				element: input.outerHTML.substring(0, 120),
				selector: cssPath(input),
				message: 'Form input missing associated label, aria-label, aria-labelledby, or title'
			});
		}
	});

	// Rule: link-name — links without accessible names
	root.querySelectorAll('a[href]').forEach(link => {
		const name = link.textContent.trim() ||
			link.getAttribute('aria-label') ||
			link.getAttribute('aria-labelledby') ||
			link.getAttribute('title');
		if (!name) {
			// Check if link contains an image with alt
			const img = link.querySelector('img[alt]');
			if (!img || !img.alt.trim()) {
				issues.push({
					severity: 'error',
					rule: 'link-name',
					element: link.outerHTML.substring(0, 120),
					selector: cssPath(link),
					message: 'Link has no accessible name'
				});
			}
		}
	});

	// Rule: landmark-unique — duplicate landmark roles without unique labels
	const landmarks = {};
	root.querySelectorAll('[role="banner"], [role="navigation"], [role="main"], [role="contentinfo"], [role="complementary"], [role="search"], nav, main, header, footer, aside').forEach(el => {
		const role = el.getAttribute('role') || el.tagName.toLowerCase();
		const label = el.getAttribute('aria-label') || el.getAttribute('aria-labelledby') || '';
		const key = role + ':' + label;
		if (!landmarks[key]) {
			landmarks[key] = [];
		}
		landmarks[key].push(el);
	});
	for (const [key, elements] of Object.entries(landmarks)) {
		if (elements.length > 1) {
			const [role, label] = key.split(':');
			if (!label) {
				issues.push({
					severity: 'warning',
					rule: 'landmark-unique',
					message: elements.length + ' ' + role + ' landmarks without unique aria-label to distinguish them',
				});
			}
		}
	}

	function cssPath(el) {
		const parts = [];
		while (el && el.nodeType === 1) {
			let part = el.tagName.toLowerCase();
			if (el.id) {
				part += '#' + el.id;
				parts.unshift(part);
				break;
			}
			const cls = Array.from(el.classList).filter(c => !c.includes(':')).slice(0, 2).join('.');
			if (cls) part += '.' + cls;
			parts.unshift(part);
			el = el.parentElement;
		}
		return parts.join(' > ');
	}

	return JSON.stringify(issues);
}
