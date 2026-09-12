// Only the test runner evaluates chrome.tabs.setZoom/getZoom in this worker.
// No content scripts, message bridge, external requests or production installation.
chrome.runtime.onInstalled.addListener(() => {});
