(function (global) {
    'use strict';

    /** 同步复制（必须在用户点击的同一调用栈内执行） */
    function copySync(text) {
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
            'z-index:2147483647',
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

    function resolveFromBtn(btn) {
        if (!btn) return '';
        const wrap = btn.closest('.yyb-acc-openid-line, .yyb-table-openid');
        const code = wrap && wrap.querySelector('.yyb-openid-text');
        if (code) {
            const t = (code.textContent || '').trim();
            if (t && t !== '-') return t;
        }
        return (btn.getAttribute('data-yyb-copy') || btn.getAttribute('data-ayyb-copy') || '').trim();
    }

    function copyWithFeedback(text, btn, onFail) {
        const s = String(text == null ? '' : text).trim();
        if (!s || s === '-') {
            if (typeof onFail === 'function') onFail('没有可复制的内容');
            return false;
        }
        if (copySync(s)) {
            if (btn) {
                btn.textContent = '已复制';
            }
            return true;
        }
        if (typeof onFail === 'function') onFail('复制失败');
        return false;
    }

    /** 捕获阶段委托，避免被账号卡片 click 抢事件 */
    function installCopyDelegation(onFail) {
        if (global.__yybCopyDelegationInstalled) return;
        global.__yybCopyDelegationInstalled = true;
        document.addEventListener('click', function (e) {
            const btn = e.target.closest('.yyb-copy-btn');
            if (!btn) return;
            if (!btn.hasAttribute('data-yyb-copy') && !btn.hasAttribute('data-ayyb-copy')) return;
            e.preventDefault();
            e.stopPropagation();
            e.stopImmediatePropagation();
            copyWithFeedback(resolveFromBtn(btn), btn, onFail);
        }, true);
    }

    function copyToClipboard(text) {
        const s = String(text == null ? '' : text);
        if (!s) return Promise.resolve(false);
        if (copySync(s)) return Promise.resolve(true);
        if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            return navigator.clipboard.writeText(s).then(() => true).catch(() => false);
        }
        return Promise.resolve(false);
    }

    global.copyToClipboardSync = copySync;
    global.copyToClipboard = copyToClipboard;
    global.YybClipboard = {
        copySync,
        copyWithFeedback,
        resolveFromBtn,
        installCopyDelegation,
    };
})(typeof window !== 'undefined' ? window : globalThis);
