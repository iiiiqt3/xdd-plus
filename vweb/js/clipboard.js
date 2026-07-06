(function (global) {
    'use strict';

    async function copyToClipboard(text) {
        const s = String(text == null ? '' : text);
        if (!s) {
            return false;
        }
        try {
            if (navigator.clipboard && global.isSecureContext) {
                await navigator.clipboard.writeText(s);
                return true;
            }
        } catch (_) { /* fallback below */ }
        try {
            const ta = document.createElement('textarea');
            ta.value = s;
            ta.setAttribute('readonly', '');
            ta.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0;pointer-events:none';
            document.body.appendChild(ta);
            ta.focus();
            ta.select();
            ta.setSelectionRange(0, s.length);
            const ok = document.execCommand('copy');
            document.body.removeChild(ta);
            return ok;
        } catch (_) {
            return false;
        }
    }

    global.copyToClipboard = copyToClipboard;
})(typeof window !== 'undefined' ? window : globalThis);
