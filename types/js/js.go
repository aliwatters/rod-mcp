package js

import _ "embed"

//go:embed snapshotter.js
var InjectedSnapShot string

//go:embed a11y_audit.js
var A11yAuditJS string

//go:embed performance_vitals.js
var PerformanceVitalsJS string

//go:embed storage_list.js
var StorageListJS string

//go:embed wait_detached.js
var WaitDetachedJS string

//go:embed smart_fill.js
var SmartFillJS string

//go:embed stealth.js
var StealthJS string

const AriaSnapshot = "function(node, opts) { return snapshotEngine.ariaSnapshot(eval(node), eval(opts)); }"

const QueryEleByAria = `(selector) => {
    return snapshotEngine.queryAll(selector);
}`
