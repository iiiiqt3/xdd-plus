(function (global) {
    'use strict';

    const state = {
        authorized: false,
        accounts: [],
        selected: new Set(),
        activeTaskId: null,
        monitorTimer: null,
        logIndex: 0,
        countdownTimer: null,
    };

    function $(id) { return document.getElementById(id); }

    function esc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    async function api(path, options) {
        const res = await fetch('/api/portal/elm' + path, {
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json', 'X-Request-Source': 'web', ...(options && options.headers || {}) },
            ...(options || {}),
        });
        if (res.redirected && res.url.includes('/portal/login')) {
            location.href = '/portal/login';
            throw new Error('未登录');
        }
        const data = await res.json();
        if (data.code !== 0) throw new Error(data.msg || '请求失败');
        return data.data;
    }

    function toast(msg, type) {
        if (typeof global.toast === 'function') global.toast(msg, type || 'info');
    }

    function elmLog(msg, level) {
        const el = $('elmLogContent');
        if (!el) return;
        const prefix = level === 'success' ? '✅ ' : level === 'error' ? '❌ ' : level === 'warn' ? '⚠️ ' : '';
        const line = `[${new Date().toLocaleTimeString('zh-CN', { hour12: false })}] ${prefix}${msg}\n`;
        el.textContent += line;
        el.scrollTop = el.scrollHeight;
    }

    function elmClearLogs() {
        const el = $('elmLogContent');
        if (el) el.textContent = '';
        state.logIndex = 0;
    }

    function renderAccounts() {
        const box = $('elmAccountList');
        if (!box) return;
        if (!state.accounts.length) {
            box.innerHTML = '<div class="muted" style="padding:12px;">暂无已上车账号，请先在「项目中心」上车饿了么(协议)活动</div>';
            return;
        }
        box.innerHTML = state.accounts.map(function (a) {
            const checked = state.selected.has(a.ref) ? 'checked' : '';
            return `<label style="display:flex;align-items:center;gap:10px;padding:10px 12px;border:1px solid var(--glass-border);border-radius:10px;background:var(--glass-bg);cursor:pointer;">
                <input type="checkbox" data-elm-ref="${esc(a.ref)}" ${checked} onchange="ElmPortal.toggleAccount(this)" style="width:16px;height:16px;accent-color:#6366f1;" />
                <div style="min-width:0;flex:1;">
                    <div style="font-weight:700;font-size:13px;color:var(--text);">${esc(a.remark || a.ref)}</div>
                    <div style="font-size:11px;color:var(--text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${esc(a.ref)}</div>
                </div>
            </label>`;
        }).join('');
    }

    function renderTodayProducts(data, win) {
        const box = $('elmProductInfo');
        if (!box) return;
        if (!data) {
            box.innerHTML = '<div class="muted">加载商品信息中…</div>';
            return;
        }
        if (!state.authorized) {
            box.innerHTML = '<div class="muted">上车饿了么(协议)活动后可查看今日抢兑商品</div>';
            return;
        }
        const star = data.starBalance >= 0 ? data.starBalance : '--';
        const slots = Array.isArray(data.slots) ? data.slots : [];
        const currentHour = win && win.canStart && win.executeAt
            ? parseInt(String(win.executeAt).slice(11, 13), 10)
            : 0;
        let html = `<div style="font-size:13px;line-height:1.8;color:var(--text-muted);margin-bottom:10px;">
            <div>💎 幸运星余额：<strong style="color:#f59e0b;">${esc(star)}</strong>${data.accountRemark ? ' · 查询账号：' + esc(data.accountRemark) : ''}</div>
            <div>${esc(data.message || '')}</div>
        </div>`;
        if (!slots.length) {
            html += '<div class="muted">暂无场次商品信息</div>';
            box.innerHTML = html;
            return;
        }
        html += '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:10px;">';
        slots.forEach(function (slot) {
            const active = currentHour === slot.targetHour;
            const border = active ? '2px solid #6366f1' : '1px solid var(--glass-border)';
            const bg = active ? 'rgba(99,102,241,0.06)' : 'var(--bg-secondary)';
            const p = slot.product;
            html += `<div style="padding:12px 14px;border-radius:10px;border:${border};background:${bg};">
                <div style="font-weight:700;color:var(--text);margin-bottom:6px;">${active ? '🎯 当前场次 · ' : ''}${esc(slot.slotLabel)}</div>
                <div style="font-size:12px;color:var(--text-muted);margin-bottom:8px;">执行时间：${esc(slot.executeAt || '-')}</div>`;
            if (p) {
                html += `<div style="font-weight:600;color:var(--text);margin-bottom:4px;">${esc(p.title)}</div>
                    <div style="font-size:12px;line-height:1.7;color:var(--text-muted);">
                        <div>所需幸运星：<strong style="color:#f59e0b;">${esc(p.cost)}</strong></div>
                        <div>状态：${esc(p.status)}</div>
                    </div>`;
            } else {
                html += '<div class="muted" style="font-size:12px;">暂未识别到该场次商品</div>';
            }
            html += '</div>';
        });
        html += '</div>';
        box.innerHTML = html;
    }

    function renderProduct(task) {
        const box = $('elmProductInfo');
        if (!box) return;
        const p = task && task.product;
        if (!p) {
            box.innerHTML = '<div class="muted">加载今日商品中…</div>';
            return;
        }
        box.innerHTML = `<div style="font-weight:700;color:var(--text);margin-bottom:6px;">🎯 ${esc(p.title)}</div>
            <div style="font-size:13px;line-height:1.8;color:var(--text-muted);">
                <div>商品 ID：<code>${esc(p.id)}</code></div>
                <div>所需幸运星：<strong style="color:#f59e0b;">${esc(p.cost)}</strong></div>
                <div>商品状态：${esc(p.status)}</div>
                <div>执行时间：<strong>${esc(task.executeAt || '-')}</strong></div>
                <div>预检时间：<strong>${esc(task.prepareAt || '-')}</strong></div>
            </div>`;
    }

    function updateWindowInfo(win) {
        const nowEl = $('elmNowTime');
        const hintEl = $('elmWindowHint');
        const nextEl = $('elmNextSlot');
        if (nowEl) {
            const d = new Date();
            nowEl.textContent = d.toLocaleString('zh-CN', { hour12: false });
        }
        if (nextEl) nextEl.textContent = win.windowLabel || '非抢兑日/时段';
        if (hintEl) {
            hintEl.textContent = win.message || '';
            hintEl.style.color = win.canStart ? '#10b981' : 'var(--text-muted)';
        }
        const btn = $('elmStartBtn');
        if (btn) btn.disabled = !win.canStart || !state.authorized;
    }

    function flushLogs(task) {
        if (!task || !Array.isArray(task.logs)) return;
        for (let i = state.logIndex; i < task.logs.length; i++) {
            const item = task.logs[i];
            elmLog((item.time ? item.time + ' ' : '') + item.message, item.level);
        }
        state.logIndex = task.logs.length;
    }

    function stopMonitor() {
        if (state.monitorTimer) {
            clearInterval(state.monitorTimer);
            state.monitorTimer = null;
        }
    }

    function stopCountdown() {
        if (state.countdownTimer) {
            clearInterval(state.countdownTimer);
            state.countdownTimer = null;
        }
        const el = $('elmCountdown');
        if (el) el.style.display = 'none';
    }

    function startCountdown(executeAt) {
        stopCountdown();
        const el = $('elmCountdown');
        if (!el || !executeAt) return;
        el.style.display = 'block';
        const target = new Date(String(executeAt).replace(/-/g, '/')).getTime();
        function tick() {
            const remain = target - Date.now();
            if (remain <= 0) {
                el.textContent = '🚀 正在执行抢兑…';
                return;
            }
            const h = Math.floor(remain / 3600000);
            const m = Math.floor((remain % 3600000) / 60000);
            const s = Math.floor((remain % 60000) / 1000);
            const ms = remain % 1000;
            el.textContent = `⏱ 倒计时 ${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}.${String(ms).padStart(3, '0')}`;
        }
        tick();
        state.countdownTimer = setInterval(tick, 100);
    }

    function renderResults(task) {
        const box = $('elmResultBox');
        if (!box || !task || !Array.isArray(task.results) || !task.results.length) return;
        box.innerHTML = task.results.map(function (r) {
            const icon = r.success ? '✅' : '❌';
            return `<div style="padding:8px 0;border-bottom:1px dashed var(--glass-border);font-size:13px;">${icon} <strong>${esc(r.remark || r.ref)}</strong> — ${esc(r.message)}${r.product ? ' · ' + esc(r.product) : ''}</div>`;
        }).join('');
    }

    function applyTask(task) {
        if (!task) return;
        state.activeTaskId = task.id;
        renderProduct(task);
        flushLogs(task);
        renderResults(task);
        if (task.status === 'pending') startCountdown(task.executeAt);
        if (task.status === 'running') {
            const el = $('elmCountdown');
            if (el) { el.style.display = 'block'; el.textContent = '🚀 正在执行抢兑…'; }
        }
        if (task.status === 'completed' || task.status === 'failed') {
            stopCountdown();
            const btn = $('elmStartBtn');
            if (btn) btn.disabled = false;
        }
    }

    function monitorTask(taskId) {
        stopMonitor();
        state.monitorTimer = setInterval(async function () {
            try {
                const task = await api('/status?taskId=' + encodeURIComponent(taskId));
                applyTask(task);
                if (task && (task.status === 'completed' || task.status === 'failed')) {
                    stopMonitor();
                    toast(task.status === 'completed' ? '抢兑任务已完成' : '抢兑任务失败', task.status === 'completed' ? 'success' : 'error');
                }
            } catch (e) {
                console.warn('elm monitor', e);
            }
        }, 1000);
    }

    async function loadTodayProducts(win) {
        if (!state.authorized) {
            renderTodayProducts(null, win);
            return;
        }
        try {
            const data = await api('/today-products');
            if (!state.activeTaskId) renderTodayProducts(data, win);
        } catch (e) {
            const box = $('elmProductInfo');
            if (box && !state.activeTaskId) {
                box.innerHTML = `<div class="muted">商品加载失败：${esc(e.message || '未知错误')}</div>`;
            }
        }
    }

    async function loadPanel() {
        try {
            const auth = await api('/check-auth');
            state.authorized = !!auth.authorized;
            const hint = $('elmAuthHint');
            if (hint) {
                hint.style.display = state.authorized ? 'none' : 'block';
                hint.textContent = auth.msg || '请先上车饿了么活动';
            }
            const win = await api('/window');
            updateWindowInfo(win);
            state.accounts = await api('/accounts') || [];
            if (state.selected.size === 0) {
                state.accounts.forEach(function (a) { state.selected.add(a.ref); });
            }
            renderAccounts();
            const active = await api('/status');
            if (active) {
                state.logIndex = 0;
                elmClearLogs();
                applyTask(active);
                monitorTask(active.id);
            } else {
                state.activeTaskId = null;
                void loadTodayProducts(win);
            }
        } catch (e) {
            toast(e.message || '加载失败', 'error');
        }
    }

    function toggleAccount(input) {
        const ref = input.getAttribute('data-elm-ref');
        if (!ref) return;
        if (input.checked) state.selected.add(ref);
        else state.selected.delete(ref);
    }

    async function startExchange() {
        if (!state.authorized) { toast('请先上车饿了么活动', 'error'); return; }
        const refs = Array.from(state.selected);
        if (!refs.length) { toast('请至少选择一个账号', 'error'); return; }
        const btn = $('elmStartBtn');
        if (btn) btn.disabled = true;
        elmClearLogs();
        state.logIndex = 0;
        try {
            const task = await api('/schedule', {
                method: 'POST',
                body: JSON.stringify({ refs: refs, keyword: '' }),
            });
            applyTask(task);
            monitorTask(task.id);
            toast('抢兑任务已创建，正在倒计时');
        } catch (e) {
            toast(e.message || '创建失败', 'error');
            if (btn) btn.disabled = false;
        }
    }

    function init() {
        const btn = $('elmStartBtn');
        if (btn) btn.onclick = function () { void startExchange(); };
        const clearBtn = $('elmClearLogBtn');
        if (clearBtn) clearBtn.onclick = elmClearLogs;
        setInterval(async function () {
            if (!$('panel-elm') || !$('panel-elm').classList.contains('active')) return;
            try {
                const win = await api('/window');
                updateWindowInfo(win);
            } catch (_) {}
        }, 1000);
    }

    global.ElmPortal = {
        init: init,
        loadPanel: loadPanel,
        toggleAccount: toggleAccount,
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})(window);
