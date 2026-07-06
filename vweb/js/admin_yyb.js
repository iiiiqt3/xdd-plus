(function (global) {
    'use strict';

    const API = '/api/admin/yyb';
    const state = { accounts: [], bindings: [], selectedOpenID: '', lastResult: '', scanSessionId: '', scanTimer: null, scanPolling: false, inited: false, tab: 'debug' };

    function $(id) { return document.getElementById(id); }

    function esc(s) { return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function request(path, opt = {}) {
        const res = await fetch(API + path, {
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json', ...(opt.headers || {}) },
            ...opt,
        });
        if (res.redirected && res.url.includes('/admin/login')) {
            location.href = '/admin/login';
            throw new Error('未登录');
        }
        const data = await res.json();
        if (data.code !== 0) throw new Error(data.msg || '失败');
        return data.data;
    }

    function accountName(a) {
        if (typeof a.nickname === 'string' && a.nickname) return a.nickname;
        return a.openid || '未命名';
    }

    function accountStatus(a) {
        const s = a.status;
        return (s && typeof s === 'object') ? '' : (s || '');
    }

    function statusBadge(s) {
        if (s === 'alive' || s === 'online') return '<span class="badge ok">可用</span>';
        return '<span class="badge bad">异常</span>';
    }

    function setResult(val, err) {
        const box = $('ayyb-resultBox');
        if (!box) return;
        const text = typeof val === 'string' ? val : JSON.stringify(val, null, 2);
        state.lastResult = text;
        box.textContent = text;
        box.classList.toggle('error', !!err);
    }

    async function copyText(text, btn) {
        const s = String(text || '');
        if (!s) {
            if (typeof global.toast === 'function') global.toast('没有可复制的内容', 'error');
            return;
        }
        const ok = typeof global.copyToClipboard === 'function'
            ? await global.copyToClipboard(s)
            : false;
        if (ok) {
            if (btn) { const p = btn.textContent; btn.textContent = '已复制'; setTimeout(() => { btn.textContent = p; }, 1200); }
            return;
        }
        if (typeof global.toast === 'function') global.toast('复制失败', 'error');
    }

    function switchTab(tab) {
        state.tab = tab;
        document.querySelectorAll('.ayyb-tab').forEach(n => n.classList.toggle('active', n.dataset.tab === tab));
        document.querySelectorAll('.ayyb-panel').forEach(p => p.classList.toggle('active', p.id === 'ayyb-panel-' + tab));
    }

    function selectedAccount() { return state.accounts.find(a => a.openid === state.selectedOpenID); }

    function syncSelected() {
        document.querySelectorAll('.ayyb-acc-opt').forEach(el => {
            el.classList.toggle('selected', el.dataset.openid === state.selectedOpenID);
        });
    }

    function renderAccountCard(acc) {
        const st = accountStatus(acc);
        const oid = esc(acc.openid);
        return `<div class="yyb-acc-card ayyb-acc-opt" data-openid="${oid}" role="button" tabindex="0">
            <div class="yyb-acc-head">
                <div class="yyb-acc-name">${esc(accountName(acc))}</div>
                ${statusBadge(st)}
            </div>
            <div class="yyb-acc-openid-line">
                <code class="yyb-openid-text" title="${oid}">${oid}</code>
                <button type="button" class="yyb-copy-btn" data-ayyb-copy="${oid}">复制</button>
            </div>
        </div>`;
    }

    function renderAccounts() {
        const box = $('ayyb-accountSelect');
        if ($('ayyb-stProtocol')) $('ayyb-stProtocol').textContent = state.accounts.length;
        if (!box) return;
        if (!state.accounts.length) {
            box.innerHTML = '<div class="yyb-empty">暂无协议账号，可点击「测试扫码」添加</div>';
            state.selectedOpenID = '';
            return;
        }
        if (!state.accounts.some(a => a.openid === state.selectedOpenID)) {
            state.selectedOpenID = state.accounts[0].openid;
        }
        box.innerHTML = state.accounts.map(renderAccountCard).join('');
        box.querySelectorAll('.ayyb-acc-opt').forEach(card => {
            card.onclick = (e) => {
                if (e.target.closest('.yyb-copy-btn')) return;
                state.selectedOpenID = card.dataset.openid;
                syncSelected();
            };
        });
        box.querySelectorAll('[data-ayyb-copy]').forEach(btn => {
            btn.onclick = (e) => { e.stopPropagation(); copyText(btn.getAttribute('data-ayyb-copy'), btn); };
        });
        syncSelected();
    }

    function renderBindings() {
        if ($('ayyb-stBindings')) $('ayyb-stBindings').textContent = state.bindings.length;
        const tbody = $('ayyb-bindingTable');
        if (!tbody) return;
        if (!state.bindings.length) {
            tbody.innerHTML = '<tr><td colspan="6" class="empty-tip">暂无绑定</td></tr>';
            return;
        }
        tbody.innerHTML = state.bindings.map(a => {
            const alive = /alive|online/i.test(a.status || '');
            const oid = esc(a.openid);
            return `<tr>
                <td>${a.userNumber}</td>
                <td>${esc(a.userNickname)}</td>
                <td>${esc(a.nickname)}</td>
                <td style="max-width:320px;">
                    <div class="yyb-table-openid">
                        <code class="yyb-openid-text" title="${oid}">${oid}</code>
                        <button type="button" class="yyb-copy-btn" data-ayyb-copy="${oid}">复制</button>
                    </div>
                </td>
                <td><span class="badge ${alive ? 'ok' : 'bad'}">${esc(a.status)}</span></td>
                <td>
                    <button class="btn btn-default" style="padding:4px 8px;font-size:12px;" data-pick="${oid}">调试</button>
                    <button class="btn btn-danger" style="padding:4px 8px;font-size:12px;" data-del="${oid}">删除</button>
                </td>
            </tr>`;
        }).join('');
        tbody.querySelectorAll('[data-ayyb-copy]').forEach(btn => {
            btn.onclick = () => copyText(btn.getAttribute('data-ayyb-copy'), btn);
        });
        tbody.querySelectorAll('[data-pick]').forEach(btn => {
            btn.onclick = () => {
                const openid = btn.getAttribute('data-pick');
                if (state.accounts.some(a => a.openid === openid)) {
                    state.selectedOpenID = openid;
                    syncSelected();
                    switchTab('debug');
                } else setResult('该账号不在协议库中', true);
            };
        });
        tbody.querySelectorAll('[data-del]').forEach(btn => {
            btn.onclick = async () => {
                const ref = btn.getAttribute('data-del');
                if (!confirm('删除该绑定及协议账号？')) return;
                try {
                    await request('/accounts/delete', { method: 'POST', body: JSON.stringify({ ref }) });
                    await loadAll();
                } catch (e) { setResult(e.message, true); }
            };
        });
    }

    async function loadStatus() {
        const st = await request('/status');
        const cfg = st.config || {};
        if ($('ayyb-stCost')) $('ayyb-stCost').textContent = cfg.scanLoginCost ?? '-';
        if ($('ayyb-stHealth')) {
            $('ayyb-stHealth').textContent = (!st.enabled || !st.ready) ? '不可用' : '正常';
            $('ayyb-stHealth').style.color = (!st.enabled || !st.ready) ? '#f56c6c' : '#67c23a';
        }
        if ($('ayyb-stBindingCount')) $('ayyb-stBindingCount').textContent = st.bindingCount ?? state.bindings.length;
        if ($('ayyb-stAliveCount')) $('ayyb-stAliveCount').textContent = st.aliveCount ?? '-';
        const disabled = !st.enabled || !st.ready;
        if ($('ayyb-scanBtn')) $('ayyb-scanBtn').disabled = disabled;
        if ($('ayyb-callBtn')) $('ayyb-callBtn').disabled = disabled;
        return st;
    }

    async function loadAll() {
        try {
            await loadStatus();
            state.accounts = await request('/protocol/accounts') || [];
            renderAccounts();
            state.bindings = await request('/accounts') || [];
            renderBindings();
        } catch (e) { setResult(e.message, true); }
    }

    async function loadDashboard(container) {
        if (!container) return;
        try {
            const st = await loadStatus();
            if (!st.enabled || !st.ready) {
                container.innerHTML = '<p style="color:#999;text-align:center;padding:16px;">' + esc(st.message || '服务不可用') + '</p>';
                return;
            }
            container.innerHTML = `
                <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:8px;">
                    <div style="text-align:center;padding:10px 6px;border-radius:8px;background:linear-gradient(135deg,rgba(64,158,255,0.08),rgba(64,158,255,0.03));border:1px solid rgba(64,158,255,0.12);">
                        <div style="font-size:22px;font-weight:800;color:#409eff;" id="ayybDashProtocol">${st.protocolCount ?? state.accounts.length}</div>
                        <div style="font-size:11px;color:#999;margin-top:2px;">协议账号</div>
                    </div>
                    <div style="text-align:center;padding:10px 6px;border-radius:8px;background:linear-gradient(135deg,rgba(103,194,58,0.08),rgba(103,194,58,0.03));border:1px solid rgba(103,194,58,0.12);">
                        <div style="font-size:22px;font-weight:800;color:#67c23a;" id="ayybDashAlive">${st.aliveCount ?? 0}</div>
                        <div style="font-size:11px;color:#999;margin-top:2px;">可用</div>
                    </div>
                    <div style="text-align:center;padding:10px 6px;border-radius:8px;background:linear-gradient(135deg,rgba(230,162,60,0.08),rgba(230,162,60,0.03));border:1px solid rgba(230,162,60,0.12);">
                        <div style="font-size:22px;font-weight:800;color:#e6a23c;">${st.bindingCount ?? state.bindings.length}</div>
                        <div style="font-size:11px;color:#999;margin-top:2px;">门户绑定</div>
                    </div>
                </div>`;
        } catch (_) {
            container.innerHTML = '<p style="color:#999;text-align:center;padding:16px;">加载失败</p>';
        }
    }

    async function withAccountAction(btn, loadingLabel, resultHint, action) {
        if (!btn || btn.disabled) return;
        const prev = btn.textContent;
        const peer = [$('ayyb-refreshBtn'), $('ayyb-resyncBtn')].filter(b => b && b !== btn);
        btn.disabled = true;
        peer.forEach(b => { b.disabled = true; });
        btn.textContent = loadingLabel;
        setResult(resultHint, false);
        try { await action(); } finally {
            btn.disabled = false;
            btn.textContent = prev;
            peer.forEach(b => { b.disabled = false; });
        }
    }

    function togglePayload() {
        const g = $('ayyb-payloadGroup');
        const sel = $('ayyb-featureSel');
        if (g && sel) g.classList.toggle('hidden', sel.value.toLowerCase() !== 'operatewxdata');
    }

    async function callFeature() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择账号', true); return; }
        const feature = $('ayyb-featureSel').value;
        const appId = $('ayyb-appidInput').value.trim();
        if (!appId) { setResult('请输入 AppID', true); return; }
        const body = { ref: acc.openid, appId };
        let path = '/wxapp/getCode';
        if (feature === 'getPhoneNumber') path = '/wxapp/getPhoneNumber';
        if (feature === 'operateWxData') {
            path = '/wxapp/operateWxData';
            try { body.payload = JSON.parse($('ayyb-payloadInput').value || '{}'); }
            catch { setResult('JSON 格式错误', true); return; }
        }
        $('ayyb-callBtn').disabled = true;
        setResult('调用中…', false);
        try {
            setResult(await request(path, { method: 'POST', body: JSON.stringify(body) }), false);
            await loadAll();
        } catch (e) { setResult(e.message, true); }
        finally { $('ayyb-callBtn').disabled = false; }
    }

    async function refreshSelected() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择协议账号', true); return; }
        try {
            await withAccountAction($('ayyb-refreshBtn'), '刷新中…', '正在刷新存活状态，请稍候…', async () => {
                setResult(await request('/accounts/refresh', { method: 'POST', body: JSON.stringify({ ref: acc.openid }) }), false);
                await loadAll();
            });
        } catch (e) { setResult(e.message, true); }
    }

    async function resyncSelected() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择协议账号', true); return; }
        try {
            await withAccountAction($('ayyb-resyncBtn'), '同步中…', '正在同步账号资料，请稍候…', async () => {
                setResult(await request('/accounts/resync', { method: 'POST', body: JSON.stringify({ ref: acc.openid }) }), false);
                await loadAll();
            });
        } catch (e) { setResult(e.message, true); }
    }

    async function deleteSelected() {
        const acc = selectedAccount();
        if (!acc || !confirm('确定删除该账号？')) return;
        try {
            await request('/accounts/delete', { method: 'POST', body: JSON.stringify({ ref: acc.openid }) });
            await loadAll();
        } catch (e) { setResult(e.message, true); }
    }

    function stopScanPoll() {
        if (state.scanTimer) { clearTimeout(state.scanTimer); state.scanTimer = null; }
        state.scanPolling = false;
    }

    function scheduleScanPoll(delay) {
        if (state.scanTimer) clearTimeout(state.scanTimer);
        state.scanTimer = setTimeout(() => { void pollScan(); }, delay);
    }

    function closeQr() {
        const m = $('ayyb-qrModal');
        if (m) m.classList.remove('show');
        stopScanPoll();
    }

    function qrImgSrc(b64) {
        if (!b64) return '';
        if (String(b64).startsWith('data:')) return b64;
        return 'data:image/png;base64,' + b64;
    }

    async function pollScan() {
        if (!state.scanSessionId || state.scanPolling) return;
        state.scanPolling = true;
        let keepPolling = false;
        try {
            const data = await request('/qr/' + encodeURIComponent(state.scanSessionId) + '/poll');
            const status = (data.status || '').toLowerCase();
            if (status === 'scanned') { $('ayyb-qrHint').textContent = '请在手机上确认登录'; keepPolling = true; return; }
            if (status === 'authorized' || status === 'confirmed') {
                stopScanPoll();
                $('ayyb-qrHint').textContent = '正在入库…';
                await request('/qr/' + encodeURIComponent(state.scanSessionId) + '/confirm', { method: 'POST' });
                closeQr();
                setResult('测试账号已入库', false);
                await loadAll();
                return;
            }
            if (['expired', 'cancelled', 'unknown'].includes(status)) {
                stopScanPoll();
                $('ayyb-qrHint').textContent = '二维码已失效';
                return;
            }
            keepPolling = true;
        } catch (e) {
            const msg = e.message || '';
            if (/deadline|timeout/i.test(msg)) { $('ayyb-qrHint').textContent = '等待扫码中（网络较慢）…'; keepPolling = true; return; }
            stopScanPoll();
            $('ayyb-qrHint').textContent = msg || '扫码失败';
            setResult(msg || '扫码确认失败', true);
        } finally {
            state.scanPolling = false;
            if (keepPolling && state.scanSessionId) scheduleScanPoll(800);
        }
    }

    async function startScan() {
        try {
            const data = await request('/qr', { method: 'POST' });
            state.scanSessionId = data.sessionId;
            const src = qrImgSrc(data.imageBase64);
            $('ayyb-qrBox').innerHTML = src ? '<img src="' + src + '" style="width:180px;height:180px;" alt="二维码">' : '无二维码';
            $('ayyb-qrModal').classList.add('show');
            stopScanPoll();
            scheduleScanPoll(0);
        } catch (e) { setResult(e.message, true); }
    }

    function bindEvents() {
        if (state.inited) return;
        state.inited = true;
        document.querySelectorAll('.ayyb-tab').forEach(btn => {
            btn.onclick = () => switchTab(btn.dataset.tab);
        });
        if ($('ayyb-featureSel')) $('ayyb-featureSel').onchange = togglePayload;
        if ($('ayyb-callBtn')) $('ayyb-callBtn').onclick = callFeature;
        if ($('ayyb-clearBtn')) $('ayyb-clearBtn').onclick = () => setResult('已清空', false);
        if ($('ayyb-copyBtn')) $('ayyb-copyBtn').onclick = async () => {
            const text = state.lastResult || ($('ayyb-resultBox') && $('ayyb-resultBox').textContent) || '';
            const ok = typeof global.copyToClipboard === 'function' && await global.copyToClipboard(text);
            if (typeof global.toast === 'function') {
                global.toast(ok ? '已复制' : '复制失败', ok ? 'success' : 'error');
            }
        };
        if ($('ayyb-refreshBtn')) $('ayyb-refreshBtn').onclick = refreshSelected;
        if ($('ayyb-resyncBtn')) $('ayyb-resyncBtn').onclick = resyncSelected;
        if ($('ayyb-deleteBtn')) $('ayyb-deleteBtn').onclick = deleteSelected;
        if ($('ayyb-scanBtn')) $('ayyb-scanBtn').onclick = startScan;
        if ($('ayyb-qrCloseBtn')) $('ayyb-qrCloseBtn').onclick = closeQr;
        togglePayload();
    }

    function init() { bindEvents(); }

    global.AdminYyb = {
        init,
        loadAll,
        loadDashboard,
        switchTab,
    };
})(window);
