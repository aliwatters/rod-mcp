package js

import _ "embed"

//go:embed snapshotter.js
var InjectedSnapShot string

//go:embed a11y_audit.js
var A11yAuditJS string

const AriaSnapshot = "function(node, opts) { return snapshotEngine.ariaSnapshot(eval(node), eval(opts)); }"

const QueryEleByAria = `(selector) => {
    return snapshotEngine.queryAll(selector);
}`
