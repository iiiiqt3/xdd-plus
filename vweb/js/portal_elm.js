(function (global) {
    'use strict';

    const state = {
        authorized: false,
        accounts: [],
        selectedRef: '',
        selectedSlot: 0,
        ckReady: false,
        ckData: null,
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

    function resetCkState() {
        state.ckReady = false;
        state.ckData = null;
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
        if (stopBtn) stopBtn.style.display = running ? 'inline-block' : 'none';
        if (fetchBtn) {
            fetchBtn.disabled = running || !state.authorized || !state.selectedRef;
        }
        if (startBtn) {
            startBtn.style.display = running ? 'none' : 'inline-block';
            const canStart = state.authorized
                && state.ckReady
                && state.selectedSlot > 0
                && state.selectedRef
                && !running;
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
            const checked = state.selectedRef === a.ref ? 'checked' : '';
            return `<label style="display:flex;align-items:center;gap:10px;padding:10px 12px;border:1px solid var(--glass-border);border-radius:10px;background:var(--glass-bg);cursor:pointer;">
                <input type="radio" name="elmAccount" data-elm-ref="${esc(a.ref)}" ${checked} onchange="ElmPortal.selectAccount(this)" style="width:16px;height:16px;accent-color:#6366f1;" />
                <div style="min-width:0;flex:1;">
                    <div style="font-weight:700;font-size:13px;color:var(--text);">${esc(a.remark || a.ref)}</div>
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
        const star = data.starBalance >= 0 ? data.starBalance : '--';
        const slots = Array.isArray(data.slots) ? data.slots : [];
        let html = `<div style="font-size:13px;line-height:1.8;color:var(--text-muted);margin-bottom:10px;">
            <div>✅ CK 已获取 · 通道：<strong>${esc(data.route || '-')}</strong></div>
            <div>💎 幸运星余额：<strong style="color:#f59e0b;">${esc(star)}</strong> · 账号：${esc(data.remark || data.ref)}</div>
            <div>${esc(data.message || '')}</div>
        </div>`;
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
        // 按钮状态由 applyTask / finishTask 维护，勿在此处重置（会与任务轮询冲突导致闪烁）
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

    function selectAccount(input) {
        const ref = input.getAttribute('data-elm-ref');
        if (!ref) return;
        if (state.selectedRef !== ref) {
            state.selectedRef = ref;
            resetCkState();
            renderTaskView(null);
        }
    }

    function selectSlot(hour) {
        state.selectedSlot = hour;
        if (state.ckData) renderCkProducts(state.ckData);
        updateActionButtons(null);
    }

    async function fetchCk() {
        if (!state.authorized) { toast('请先上车饿了么活动', 'error'); return; }
        if (!state.selectedRef) { toast('请先选择一个账号', 'error'); return; }
        const btn = $('elmFetchCkBtn');
        if (btn) { btn.disabled = true; btn.textContent = '获取中…'; }
        resetCkState();
        state.selectedRef = state.selectedRef || '';
        try {
            const data = await api('/fetch-ck', {
                method: 'POST',
                body: JSON.stringify({ ref: state.selectedRef }),
            });
            state.ckReady = true;
            state.ckData = data;
            const slotPicker = $('elmSlotPicker');
            if (slotPicker) slotPicker.style.display = 'block';
            renderCkProducts(data);
            elmLog('CK 获取成功：' + (data.remark || data.ref) + ' · ' + (data.route || ''), 'success');
            toast('CK 获取成功，请选择场次', 'success');
        } catch (e) {
            state.ckReady = false;
            state.ckData = null;
            const box = $('elmProductInfo');
            if (box) {
                box.innerHTML = `<div style="color:#dc2626;font-size:13px;line-height:1.7;">
                    ❌ CK 获取失败：${esc(e.message || '未知错误')}<br>
                    <span class="muted">请检查账号是否在线，或稍后点击「获取 CK」重试</span>
                </div>`;
            }
            elmLog('CK 获取失败：' + (e.message || '未知错误'), 'error');
            toast(e.message || 'CK 获取失败，请重试', 'error');
        } finally {
            if (btn) { btn.textContent = '获取 CK'; updateActionButtons(null); }
        }
    }

    async function startExchange() {
        if (!state.authorized) { toast('请先上车饿了么活动', 'error'); return; }
        if (!state.selectedRef) { toast('请选择一个账号', 'error'); return; }
        if (!state.ckReady) { toast('请先获取 CK', 'error'); return; }
        if (!state.selectedSlot) { toast('请选择 10 点或 15 点场次', 'error'); return; }
        elmClearLogs();
        state.logIndex = 0;
        updateActionButtons({ status: 'pending' });
        try {
            const task = await api('/schedule', {
                method: 'POST',
                body: JSON.stringify({
                    ref: state.selectedRef,
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
        selectAccount: selectAccount,
        selectSlot: selectSlot,
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})(window);
