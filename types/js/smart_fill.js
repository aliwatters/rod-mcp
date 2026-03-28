// smart_fill.js — React-aware fill strategy.
// Returns a JSON stringified object: { "method": "<method used>", "value": "<final value>", "react": <boolean>, "success": <boolean> }
// Accepts (value) via rod's Eval binding; the element is bound as `this`.
(function(value) {
    "use strict";
    var el = this;

    // Helper: check if element is a React controlled input.
    function isReactControlled(el) {
        var keys = Object.keys(el);
        for (var i = 0; i < keys.length; i++) {
            if (keys[i].indexOf("__reactFiber$") === 0 ||
                keys[i].indexOf("__reactProps$") === 0 ||
                keys[i].indexOf("__reactInternalInstance$") === 0) {
                return true;
            }
        }
        return false;
    }

    // Helper: get the current value of the element.
    function getValue(el) {
        return el.value !== undefined ? el.value : el.textContent;
    }

    // Helper: scroll into view and focus.
    function prepareElement(el) {
        el.scrollIntoView({ block: "center" });
        el.focus();
    }

    // Strategy 1: Standard input (works for non-React inputs).
    function tryStandardInput(el, value) {
        var nativeInputValueSetter = Object.getOwnPropertyDescriptor(
            window.HTMLInputElement.prototype, "value"
        );
        var nativeTextAreaValueSetter = Object.getOwnPropertyDescriptor(
            window.HTMLTextAreaElement.prototype, "value"
        );

        var setter = null;
        if (el instanceof HTMLInputElement && nativeInputValueSetter && nativeInputValueSetter.set) {
            setter = nativeInputValueSetter.set;
        } else if (el instanceof HTMLTextAreaElement && nativeTextAreaValueSetter && nativeTextAreaValueSetter.set) {
            setter = nativeTextAreaValueSetter.set;
        }

        if (setter) {
            setter.call(el, value);
        } else {
            el.value = value;
        }

        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("change", { bubbles: true }));

        return getValue(el) === value;
    }

    // Strategy 2: ClipboardEvent + execCommand (works for React controlled inputs).
    function tryClipboardPaste(el, value) {
        // Select all existing text first.
        if (el.select) { el.select(); }
        else if (el.setSelectionRange) {
            el.setSelectionRange(0, (el.value || el.textContent || "").length);
        }

        // Try execCommand insertText first (works in most browsers).
        if (document.execCommand("insertText", false, value)) {
            if (getValue(el) === value) return true;
        }

        // Fallback: synthetic ClipboardEvent paste.
        if (el.select) { el.select(); }
        var dt = new DataTransfer();
        dt.setData("text/plain", value);
        var pasteEvent = new ClipboardEvent("paste", {
            clipboardData: dt,
            bubbles: true,
            cancelable: true
        });
        el.dispatchEvent(pasteEvent);
        document.execCommand("insertText", false, value);

        return getValue(el) === value;
    }

    // Strategy 3: Simulated key-by-key input events (last resort).
    function tryKeyByKey(el, value) {
        // Clear existing value.
        if (el.select) { el.select(); }
        document.execCommand("delete", false, null);

        for (var i = 0; i < value.length; i++) {
            var ch = value[i];
            el.dispatchEvent(new KeyboardEvent("keydown", { key: ch, bubbles: true }));
            el.dispatchEvent(new KeyboardEvent("keypress", { key: ch, bubbles: true }));
            document.execCommand("insertText", false, ch);
            el.dispatchEvent(new KeyboardEvent("keyup", { key: ch, bubbles: true }));
        }

        return getValue(el) === value;
    }

    // Main logic.
    prepareElement(el);

    var reactControlled = isReactControlled(el);
    var method = "";

    if (!reactControlled) {
        // Non-React: try standard first.
        if (tryStandardInput(el, value)) {
            method = "standard";
        } else if (tryClipboardPaste(el, value)) {
            method = "clipboard_fallback";
        } else if (tryKeyByKey(el, value)) {
            method = "keybykey_fallback";
        }
    } else {
        // React: skip standard, go straight to clipboard.
        if (tryClipboardPaste(el, value)) {
            method = "react_clipboard";
        } else if (tryKeyByKey(el, value)) {
            method = "react_keybykey";
        } else if (tryStandardInput(el, value)) {
            method = "react_standard_fallback";
        }
    }

    var finalValue = getValue(el);
    if (method === "") {
        method = "none_succeeded";
    }

    return JSON.stringify({
        method: method,
        value: finalValue,
        react: reactControlled,
        success: finalValue === value
    });
})
