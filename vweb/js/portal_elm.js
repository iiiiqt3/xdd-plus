(function (global) {
    'use strict';

    const state = {
        authorized: false,
        accounts: [],
        selected: new Set(),
        selectedSlot: 0,
        ckReady: false,
        ckData: null,
        ckReadyRefs: new Set(),
        activeTaskId: null,
        monitorTimer: null,
        logIndex: 0,
        countdownTimer: null,
        window: null,
    };

    function $(id) { return document.getElementById(id); }

    function esc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    function selectedRefs() {
        return Array.from(state.selected);
    }

    function isBatchCk(data) {
        return !!(data && Array.isArray(data.accounts));
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

    function syncCkReady() {
        const refs = selectedRefs();
        state.ckReady = refs.length > 0 && refs.every(function (r) { return state.ckReadyRefs.has(r); });
    }

    function resetCkState() {
        state.ckReady = false;
        state.ckData = null;
        state.ckReadyRefs = new Set();
        state.selectedSlot = 0;
        const slotPicker = $('elmSlotPicker');
        if (slotPicker) slotPicker.style.display = 'none';
        document.querySelectorAll('input[name="elmSlot"]').forEach(function (el) { el.checked = false; });
        updateActionButtons();
    }

    function updateActionButtons(task) {
        const startBtn = $('elmStartBtn');
        const stopBtn = $('elmStopBtn');
        const fetchBtn = $('elmFetchCkBtn');
        const running = !!(
            state.activeTaskId ||
            (task && (task.status === 'pending' || task.status === 'running'))
        );
        const hasSelection = state.selected.size > 0;
        if (stopBtn) stopBtn.style.display = running ? 'inline-block' : 'none';
        if (fetchBtn) {
            fetchBtn.disabled = running || !state.authorized || !hasSelection;
        }
        if (startBtn) {
            startBtn.style.display = running ? 'none' : 'inline-block';
            const canStart = state.authorized
                && state.ckReady
                && state.selectedSlot > 0
                && hasSelection
                && !running
                && state.window
                && state.window.canStart;
            startBtn.disabled = !canStart;
            startBtn.style.opacity = canStart ? '1' : '0.55';
            startBtn.style.cursor = canStart ? 'pointer' : 'not-allowed';
        }
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
            const ckOk = state.ckReadyRefs.has(a.ref) ? ' · ✅CK' : '';
            return `<label style="display:flex;align-items:center;gap:10px;padding:10px 12px;border:1px solid var(--glass-border);border-radius:10px;background:var(--glass-bg);cursor:pointer;">
                <input type="checkbox" data-elm-ref="${esc(a.ref)}" ${checked} onchange="ElmPortal.toggleAccount(this)" style="width:16px;height:16px;accent-color:#6366f1;" />
                <div style="min-width:0;flex:1;">
                    <div style="font-weight:700;font-size:13px;color:var(--text);">${esc(a.remark || a.ref)}${ckOk}</div>
                    <div style="font-size:11px;color:var(--text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${esc(a.ref)}</div>
                </div>
            </label>`;
        }).join('');
    }

    function renderCkProducts(data) {
        const box = $('elmProductInfo');
        if (!box) return;
        if (!data) {
            box.innerHTML = '<div class="muted">请先选择账号并点击「获取 CK」</div>';
            return;
        }

        let html = '';
        if (isBatchCk(data)) {
            html += `<div style="font-size:13px;line-height:1.8;color:var(--text-muted);margin-bottom:10px;">
                <div>CK 就绪：<strong>${esc(data.readyCount)}/${esc(data.totalCount)}</strong></div>
                <div>${esc(data.message || '')}</div>
            </div>`;
            html += '<div style="margin-bottom:12px;">';
            data.accounts.forEach(function (item) {
                const icon = item.success ? '✅' : '❌';
                html += `<div style="font-size:12px;padding:6px 0;border-bottom:1px dashed var(--glass-border);">${icon} <strong>${esc(item.remark || item.ref)}</strong> — ${esc(item.message || '')}${item.route ? ' · ' + esc(item.route) : ''}${item.success && item.starBalance >= 0 ? ' · ' + esc(item.starBalance) + '星' : ''}</div>`;
            });
            html += '</div>';
            renderSlotsHtml(html, data.slots || [], box);
            return;
        }

        const star = data.starBalance >= 0 ? data.starBalance : '--';
        html += `<div style="font-size:13px;line-height:1.8;color:var(--text-muted);margin-bottom:10px;">
            <div>✅ CK 已获取 · 通道：<strong>${esc(data.route || '-')}</strong></div>
            <div>💎 幸运星余额：<strong style="color:#f59e0b;">${esc(star)}</strong> · 账号：${esc(data.remark || data.ref)}</div>
            <div>${esc(data.message || '')}</div>
        </div>`;
        renderSlotsHtml(html, data.slots || [], box);
    }

    function renderSlotsHtml(prefix, slots, box) {
        let html = prefix;
        if (!slots.length) {
            html += '<div class="muted">暂无场次商品信息</div>';
            box.innerHTML = html;
            return;
        }
        html += '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:10px;">';
        slots.forEach(function (slot) {
            const active = state.selectedSlot === slot.targetHour;
            const border = active ? '2px solid #6366f1' : '1px solid var(--glass-border)';
            const bg = active ? 'rgba(99,102,241,0.06)' : 'var(--bg-secondary)';
            const p = slot.product;
            html += `<div style="padding:12px 14px;border-radius:10px;border:${border};background:${bg};">
                <div style="font-weight:700;color:var(--text);margin-bottom:6px;">${active ? '🎯 已选 · ' : ''}${esc(slot.slotLabel)}</div>
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
        if (!box || !task || !task.product) return;
        const p = task.product;
        box.innerHTML = `<div style="font-weight:700;color:var(--text);margin-bottom:6px;">🎯 ${esc(p.title)}</div>
            <div style="font-size:13px;line-height:1.8;color:var(--text-muted);">
                <div>商品 ID：<code>${esc(p.id)}</code></div>
                <div>所需幸运星：<strong style="color:#f59e0b;">${esc(p.cost)}</strong></div>
                <div>商品状态：${esc(p.status)}</div>
                <div>执行时间：<strong>${esc(task.executeAt || '-')}</strong></div>
                <div>预检时间：<strong>${esc(task.prepareAt || '-')}</strong></div>
            </div>`;
    }

    function renderTaskView(task) {
        if (task && task.product) {
            renderProduct(task);
            return;
        }
        if (state.ckData) {
            renderCkProducts(state.ckData);
            return;
        }
        const box = $('elmProductInfo');
        if (box) box.innerHTML = '<div class="muted">请先选择账号并点击「获取 CK」</div>';
    }

    function updateWindowInfo(win) {
        state.window = win;
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
        if (!box) return;
        if (!task || !Array.isArray(task.results) || !task.results.length) {
            box.innerHTML = '<div class="muted">暂无结果</div>';
            return;
        }
        box.innerHTML = task.results.map(function (r) {
            const icon = r.success ? '✅' : '❌';
            return `<div style="padding:8px 0;border-bottom:1px dashed var(--glass-border);font-size:13px;">${icon} <strong>${esc(r.remark || r.ref)}</strong> — ${esc(r.message)}${r.product ? ' · ' + esc(r.product) : ''}</div>`;
        }).join('');
    }

    function finishTask(task, toastMsg, toastType) {
        stopMonitor();
        stopCountdown();
        state.activeTaskId = null;
        applyTask(task);
        updateActionButtons(null);
        if (toastMsg) toast(toastMsg, toastType || 'info');
    }

    function applyTask(task) {
        if (!task) return;
        state.activeTaskId = (task.status === 'pending' || task.status === 'running') ? task.id : null;
        renderTaskView(task);
        flushLogs(task);
        renderResults(task);
        updateActionButtons(task);
        if (task.status === 'pending') startCountdown(task.executeAt);
        if (task.status === 'running') {
            const el = $('elmCountdown');
            if (el) { el.style.display = 'block'; el.textContent = '🚀 正在执行抢兑…'; }
        }
        if (task.status === 'completed' || task.status === 'failed' || task.status === 'cancelled') {
            stopCountdown();
        }
    }

    function monitorTask(taskId) {
        stopMonitor();
        state.monitorTimer = setInterval(async function () {
            try {
                const task = await api('/status?taskId=' + encodeURIComponent(taskId));
                applyTask(task);
                if (task && (task.status === 'completed' || task.status === 'failed' || task.status === 'cancelled')) {
                    const msg = task.status === 'completed' ? '抢兑任务已完成'
                        : task.status === 'cancelled' ? '抢兑任务已停止' : '抢兑任务失败';
                    const type = task.status === 'completed' ? 'success' : task.status === 'cancelled' ? 'info' : 'error';
                    finishTask(task, msg, type);
                }
            } catch (e) {
                console.warn('elm monitor', e);
            }
        }, 1000);
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
            renderAccounts();
            renderTaskView(null);
            const active = await api('/status');
            if (active && (active.status === 'pending' || active.status === 'running')) {
                state.logIndex = 0;
                elmClearLogs();
                applyTask(active);
                monitorTask(active.id);
            } else {
                state.activeTaskId = null;
                updateActionButtons(null);
            }
        } catch (e) {
            toast(e.message || '加载失败', 'error');
        }
    }

    function toggleAccount(input) {
        const ref = input.getAttribute('data-elm-ref');
        if (!ref) return;
        if (input.checked) state.selected.add(ref);
        else {
            state.selected.delete(ref);
            state.ckReadyRefs.delete(ref);
        }
        syncCkReady();
        renderAccounts();
        if (!state.selected.size) {
            resetCkState();
            renderTaskView(null);
        } else if (!state.ckReady) {
            state.ckData = null;
            state.selectedSlot = 0;
            const slotPicker = $('elmSlotPicker');
            if (slotPicker) slotPicker.style.display = 'none';
            document.querySelectorAll('input[name="elmSlot"]').forEach(function (el) { el.checked = false; });
            renderTaskView(null);
        }
        updateActionButtons();
    }

    function selectSlot(hour) {
        state.selectedSlot = hour;
        if (state.ckData) renderCkProducts(state.ckData);
        updateActionButtons(null);
    }

    function applyFetchResult(data) {
        state.ckData = data;
        state.ckReadyRefs = new Set();
        if (isBatchCk(data)) {
            data.accounts.forEach(function (item) {
                if (item.success) state.ckReadyRefs.add(item.ref);
            });
        } else if (data && data.ref) {
            state.ckReadyRefs.add(data.ref);
        }
        syncCkReady();
        const slotPicker = $('elmSlotPicker');
        if (slotPicker) slotPicker.style.display = state.ckReady ? 'block' : 'none';
        renderCkProducts(data);
        renderAccounts();
    }

    async function fetchCk() {
        if (!state.authorized) { toast('请先上车饿了么活动', 'error'); return; }
        const refs = selectedRefs();
        if (!refs.length) { toast('请至少选择一个账号', 'error'); return; }
        const btn = $('elmFetchCkBtn');
        if (btn) { btn.disabled = true; btn.textContent = '获取中…'; }
        state.ckReady = false;
        state.ckData = null;
        state.ckReadyRefs = new Set();
        state.selectedSlot = 0;
        document.querySelectorAll('input[name="elmSlot"]').forEach(function (el) { el.checked = false; });
        try {
            const data = await api('/fetch-ck', {
                method: 'POST',
                body: JSON.stringify({ refs: refs }),
            });
            applyFetchResult(data);
            if (isBatchCk(data)) {
                data.accounts.forEach(function (item) {
                    elmLog((item.success ? 'CK 成功：' : 'CK 失败：') + (item.remark || item.ref) + ' — ' + (item.message || ''), item.success ? 'success' : 'error');
                });
            } else {
                elmLog('CK 获取成功：' + (data.remark || data.ref) + ' · ' + (data.route || ''), 'success');
            }
            if (state.ckReady) {
                toast(data.message || 'CK 获取成功，请选择场次', 'success');
            } else {
                toast(data.message || '部分账号 CK 失败，请重试', 'warn');
            }
        } catch (e) {
            state.ckReady = false;
            state.ckData = null;
            state.ckReadyRefs = new Set();
            const box = $('elmProductInfo');
            if (box) {
                box.innerHTML = `<div style="color:#dc2626;font-size:13px;line-height:1.7;">
                    ❌ CK 获取失败：${esc(e.message || '未知错误')}<br>
                    <span class="muted">请检查账号是否在线，或稍后点击「获取 CK」重试</span>
                </div>`;
            }
            elmLog('CK 获取失败：' + (e.message || '未知错误'), 'error');
            toast(e.message || 'CK 获取失败，请重试', 'error');
            renderAccounts();
        } finally {
            if (btn) { btn.textContent = '获取 CK'; updateActionButtons(null); }
        }
    }

    async function startExchange() {
        if (!state.authorized) { toast('请先上车饿了么活动', 'error'); return; }
        const refs = selectedRefs();
        if (!refs.length) { toast('请至少选择一个账号', 'error'); return; }
        if (!state.ckReady) { toast('请先为全部选中账号获取 CK', 'error'); return; }
        if (!state.selectedSlot) { toast('请选择 10 点或 15 点场次', 'error'); return; }
        elmClearLogs();
        state.logIndex = 0;
        updateActionButtons({ status: 'pending' });
        try {
            const task = await api('/schedule', {
                method: 'POST',
                body: JSON.stringify({
                    refs: refs,
                    targetHour: state.selectedSlot,
                    keyword: '',
                }),
            });
            applyTask(task);
            monitorTask(task.id);
            toast('抢兑任务已创建，正在倒计时');
        } catch (e) {
            toast(e.message || '创建失败', 'error');
            updateActionButtons(null);
        }
    }

    async function stopExchange() {
        const stopBtn = $('elmStopBtn');
        if (stopBtn) stopBtn.disabled = true;
        try {
            const task = await api('/cancel', {
                method: 'POST',
                body: JSON.stringify({ taskId: state.activeTaskId || '' }),
            });
            finishTask(task, '抢兑任务已停止', 'info');
        } catch (e) {
            toast(e.message || '停止失败', 'error');
        } finally {
            if (stopBtn) stopBtn.disabled = false;
        }
    }

    function init() {
        const fetchBtn = $('elmFetchCkBtn');
        if (fetchBtn) fetchBtn.onclick = function () { void fetchCk(); };
        const btn = $('elmStartBtn');
        if (btn) btn.onclick = function () { void startExchange(); };
        const stopBtn = $('elmStopBtn');
        if (stopBtn) stopBtn.onclick = function () { void stopExchange(); };
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
        selectSlot: selectSlot,
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})(window);
