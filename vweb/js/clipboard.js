(function (global) {
    'use strict';

    /** 同步复制（必须在 click 回调里直接调用，不能 await 之后再调） */
    function copyToClipboardSync(text) {
        const s = String(text == null ? '' : text);
        if (!s) {
            return false;
        }
        const active = document.activeElement;
        const ta = document.createElement('textarea');
        ta.value = s;
        ta.setAttribute('readonly', '');
        ta.style.cssText = [
            'position:fixed',
            'top:0',
            'left:0',
            'width:2em',
            'height:2em',
            'padding:0',
            'border:none',
            'outline:none',
            'box-shadow:none',
            'background:transparent',
            'opacity:0',
            'z-index:-1',
        ].join(';');
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        ta.setSelectionRange(0, s.length);
        let ok = false;
        try {
            ok = document.execCommand('copy');
        } catch (_) {
            ok = false;
        }
        document.body.removeChild(ta);
        if (active && typeof active.focus === 'function') {
            try { active.focus(); } catch (_) { /* ignore */ }
        }
        return ok;
    }

    /** 先同步复制，失败再尝试 Clipboard API */
    function copyToClipboard(text) {
        const s = String(text == null ? '' : text);
        if (!s) {
            return Promise.resolve(false);
        }
        if (copyToClipboardSync(s)) {
            return Promise.resolve(true);
        }
        if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            return navigator.clipboard.writeText(s).then(() => true).catch(() => false);
        }
        return Promise.resolve(false);
    }

    global.copyToClipboardSync = copyToClipboardSync;
    global.copyToClipboard = copyToClipboard;
})(typeof window !== 'undefined' ? window : globalThis);
