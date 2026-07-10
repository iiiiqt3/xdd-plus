(function (global) {
    'use strict';

    const API = '/api/admin/yyb';
    const state = { bindings: [], protocolCount: 0, aliveCount: 0, lastResult: '', scanSessionId: '', scanTimer: null, scanPolling: false, inited: false, tab: 'debug', bindingLoading: false, warmupStarted: false, statusPollTimer: null };

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

    function statusTag(s) {
        const st = String(s || '').toLowerCase();
        if (st === 'alive' || st === 'online') return '<span class="wx-device-badge online-badge">🟢 可用</span>';
        if (st === 'dead' || st === 'offline' || st === 'expired') return '<span class="wx-device-badge offline-badge">🔴 失效</span>';
        return '<span class="wx-device-badge secondary-badge">' + esc(s || '未知') + '</span>';
    }

    function setResult(val, err) {
        const box = $('ayyb-resultBox');
        if (!box) return;
        const text = typeof val === 'string' ? val : JSON.stringify(val, null, 2);
        state.lastResult = text;
        box.textContent = text;
        box.classList.toggle('error', !!err);
    }

    function copyFail(msg) {
        if (typeof global.toast === 'function') global.toast(msg, 'error');
    }

    function copyText(text, btn) {
        if (global.YybClipboard) {
            return global.YybClipboard.copyWithFeedback(text, btn, copyFail);
        }
        return global.copyToClipboardSync && global.copyToClipboardSync(String(text || ''));
    }

    function setupCopy() {
        if (global.YybClipboard) {
            global.YybClipboard.installCopyDelegation(copyFail);
        }
    }

    function attrEsc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/"/g, '&quot;');
    }

    function switchTab(tab) {
        state.tab = tab;
        document.querySelectorAll('.ayyb-tab').forEach(n => n.classList.toggle('active', n.dataset.tab === tab));
        document.querySelectorAll('.ayyb-panel').forEach(p => p.classList.toggle('active', p.id === 'ayyb-panel-' + tab));
    }

    function formatUin(acc) {
        const u = acc && acc.uin;
        if (u != null && u !== '' && Number(u) > 0) return String(u);
        return '';
    }

    function setBindingStatus(text, type) {
        const el = $('ayyb-bindingStatus');
        if (!el) return;
        el.className = 'ayyb-binding-status' + (type ? (' ' + type) : '');
        el.textContent = text || '';
    }

    function formatCheckSummary(s) {
        if (!s) return '';
        const alive = Number(s.alive || 0);
        const dead = Number(s.dead || 0);
        const failed = Number(s.failed || 0);
        const total = Number(s.total || 0);
        if (!total) return '暂无绑定账号';
        let msg = '检测完成：共 ' + total + ' 个，可用 ' + alive + ' 个';
        if (dead > 0) msg += '，失效 ' + dead + ' 个';
        if (failed > 0) msg += '，失败 ' + failed + ' 个';
        return msg;
    }

    function renderBindings() {
        if ($('ayyb-stBindings')) $('ayyb-stBindings').textContent = state.bindings.length;
        const tbody = $('ayyb-bindingTable');
        if (!tbody) return;
        if (!state.bindings.length) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-tip">暂无绑定</td></tr>';
            return;
        }
        tbody.innerHTML = state.bindings.map(a => {
            const rawOpenid = String(a.openid || '');
            const oid = esc(rawOpenid);
            const uin = formatUin(a);
            return `<tr>
                <td>${a.userNumber}</td>
                <td>${esc(a.userNickname)}</td>
                <td>${esc(a.nickname)}</td>
                <td style="max-width:140px;font-family:Consolas,monospace;font-size:12px;" title="${uin ? attrEsc(uin) : '扫码绑定后自动获取'}">${esc(uin || '未获取')}</td>
                <td style="max-width:320px;">
                    <div class="yyb-table-openid">
                        <code class="yyb-openid-text" title="${attrEsc(rawOpenid)}">${oid}</code>
                        <button type="button" class="yyb-copy-btn" data-ayyb-copy="${attrEsc(rawOpenid)}">复制</button>
                    </div>
                </td>
                <td>${statusTag(a.status)}</td>
                <td>
                    <button class="btn btn-default" style="padding:4px 8px;font-size:12px;" data-pick="${attrEsc(rawOpenid)}">调试</button>
                    <button class="btn btn-danger" style="padding:4px 8px;font-size:12px;" data-del="${attrEsc(rawOpenid)}">删除</button>
                </td>
            </tr>`;
        }).join('');
        tbody.querySelectorAll('[data-pick]').forEach(btn => {
            btn.onclick = () => {
                const openid = btn.getAttribute('data-pick');
                if ($('ayyb-refInput')) $('ayyb-refInput').value = openid;
                switchTab('debug');
                if (typeof global.toast === 'function') global.toast('已填入 OpenID，可前往调试台执行调用', 'success');
            };
        });
        tbody.querySelectorAll('[data-del]').forEach(btn => {
            btn.onclick = async () => {
                const ref = btn.getAttribute('data-del');
                if (!confirm('删除该绑定及协议账号？')) return;
                try {
                    await request('/accounts/delete', { method: 'POST', body: JSON.stringify({ ref }) });
                    await loadBindings(false);
                    await loadStatus();
                } catch (e) { setResult(e.message, true); }
            };
        });
    }

    const WARMUP_KEY = 'xdd_admin_yyb_proto_warmup';

    function applyAliveCount(n) {
        state.aliveCount = Number(n || 0);
        if ($('ayyb-stAlive')) $('ayyb-stAlive').textContent = state.aliveCount;
        if ($('ayybDashAlive')) $('ayybDashAlive').textContent = state.aliveCount;
    }

    async function loadStatus() {
        const st = await request('/status');
        const cfg = st.config || {};
        if ($('ayyb-stCost')) {
            const cost = cfg.scanLoginCost;
            $('ayyb-stCost').textContent = (cost == null ? 2000 : cost);
        }
        if ($('ayyb-stHealth')) {
            $('ayyb-stHealth').textContent = (!st.enabled || !st.ready) ? '不可用' : '正常';
            $('ayyb-stHealth').style.color = (!st.enabled || !st.ready) ? '#f56c6c' : '#67c23a';
        }
        if ($('ayyb-stProtocol')) $('ayyb-stProtocol').textContent = st.protocolCount ?? state.protocolCount;
        if (st.protocolCount != null) state.protocolCount = st.protocolCount;
        applyAliveCount(st.aliveCount ?? state.aliveCount);
        if ($('ayyb-stBindings') && st.bindingCount != null) $('ayyb-stBindings').textContent = st.bindingCount;
        const disabled = !st.enabled || !st.ready;
        if ($('ayyb-scanBtn')) $('ayyb-scanBtn').disabled = disabled;
        if ($('ayyb-callBtn')) $('ayyb-callBtn').disabled = disabled;
        if ($('ayyb-checkAllBtn')) $('ayyb-checkAllBtn').disabled = disabled;
        return st;
    }

    function stopStatusPoll() {
        if (state.statusPollTimer) {
            clearTimeout(state.statusPollTimer);
            state.statusPollTimer = null;
        }
    }

    function scheduleStatusPoll(delay) {
        stopStatusPoll();
        state.statusPollTimer = setTimeout(async () => {
            try {
                const st = await loadStatus();
                if (st && st.checkRunning) scheduleStatusPoll(2500);
            } catch (_) { /* ignore */ }
        }, delay);
    }

    // 每次管理员登录会话只自动检测 1 次（切页/再进应用宝不再触发）
    async function warmupProtocolCheck() {
        if (state.warmupStarted) return;
        try {
            if (sessionStorage.getItem(WARMUP_KEY) === '1') {
                state.warmupStarted = true;
                return;
            }
        } catch (_) { /* ignore */ }
        state.warmupStarted = true;
        try {
            sessionStorage.setItem(WARMUP_KEY, '1');
        } catch (_) { /* ignore */ }
        try {
            const data = await request('/protocol/warmup', { method: 'POST', body: '{}' });
            if (data && data.started) scheduleStatusPoll(2000);
        } catch (_) { /* 静默：不影响其他页面 */ }
    }

    async function loadBindings(showToast) {
        if (state.bindingLoading) return;
        state.bindingLoading = true;
        setBindingStatus('正在刷新绑定列表…', 'loading');
        try {
            state.bindings = await request('/accounts') || [];
            renderBindings();
            setBindingStatus('共 ' + state.bindings.length + ' 个绑定账号', 'ok');
            if (showToast && typeof global.toast === 'function') global.toast('绑定列表已刷新', 'success');
        } catch (e) {
            setBindingStatus(e.message || '加载失败', 'warn');
            if (typeof global.toast === 'function') global.toast(e.message || '加载失败', 'error');
        } finally {
            state.bindingLoading = false;
        }
    }

    async function checkAllBindings() {
        if (state.bindingLoading) return;
        state.bindingLoading = true;
        const btn = $('ayyb-checkAllBtn');
        const prev = btn ? btn.textContent : '';
        if (btn) { btn.disabled = true; btn.textContent = '检测中…'; }
        setBindingStatus('正在检测全部账号存活状态，请稍候…', 'loading');
        try {
            const data = await request('/accounts/check-all', { method: 'POST', body: '{}' });
            state.bindings = data.accounts || [];
            renderBindings();
            const summaryText = formatCheckSummary(data.checkSummary);
            const dead = Number((data.checkSummary || {}).dead || 0);
            const failed = Number((data.checkSummary || {}).failed || 0);
            setBindingStatus(summaryText, (dead > 0 || failed > 0) ? 'warn' : 'ok');
            await loadStatus();
            if (typeof global.toast === 'function') global.toast(summaryText, dead > 0 ? 'error' : 'success');
        } catch (e) {
            setBindingStatus(e.message || '检测失败', 'warn');
            if (typeof global.toast === 'function') global.toast(e.message || '检测失败', 'error');
        } finally {
            state.bindingLoading = false;
            if (btn) { btn.disabled = false; btn.textContent = prev; }
        }
    }

    async function loadAll() {
        try {
            await loadStatus();
            const protocol = await request('/protocol/accounts') || [];
            state.protocolCount = protocol.length;
            if ($('ayyb-stProtocol')) $('ayyb-stProtocol').textContent = state.protocolCount;
            const alive = protocol.filter(a => {
                const st = String((a && a.status) || '').toLowerCase();
                return st === 'alive' || st === 'online';
            }).length;
            applyAliveCount(alive);
            await loadBindings(false);
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
                        <div style="font-size:22px;font-weight:800;color:#409eff;" id="ayybDashProtocol">${st.protocolCount ?? state.protocolCount}</div>
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

    function togglePayload() {
        const g = $('ayyb-payloadGroup');
        const sel = $('ayyb-featureSel');
        if (g && sel) g.classList.toggle('hidden', sel.value.toLowerCase() !== 'operatewxdata');
    }

    async function callFeature() {
        const ref = ($('ayyb-refInput') && $('ayyb-refInput').value || '').trim();
        if (!ref) { setResult('请输入 OpenID 或账号 Ref', true); return; }
        const feature = $('ayyb-featureSel').value;
        const appId = $('ayyb-appidInput').value.trim();
        if (!appId) { setResult('请输入 AppID', true); return; }
        const body = { ref, appId };
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
        } catch (e) { setResult(e.message, true); }
        finally { $('ayyb-callBtn').disabled = false; }
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
                const result = await request('/qr/' + encodeURIComponent(state.scanSessionId) + '/confirm', { method: 'POST' });
                closeQr();
                setResult('测试账号已入库', false);
                if (result && result.account && result.account.openid && $('ayyb-refInput')) {
                    $('ayyb-refInput').value = result.account.openid;
                }
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
        setupCopy();
        document.querySelectorAll('.ayyb-tab').forEach(btn => {
            btn.onclick = () => switchTab(btn.dataset.tab);
        });
        if ($('ayyb-featureSel')) $('ayyb-featureSel').onchange = togglePayload;
        if ($('ayyb-callBtn')) $('ayyb-callBtn').onclick = callFeature;
        if ($('ayyb-clearBtn')) $('ayyb-clearBtn').onclick = () => setResult('已清空', false);
        if ($('ayyb-copyBtn')) $('ayyb-copyBtn').onclick = () => {
            const text = state.lastResult || ($('ayyb-resultBox') && $('ayyb-resultBox').textContent) || '';
            copyText(text, $('ayyb-copyBtn'));
        };
        if ($('ayyb-checkAllBtn')) $('ayyb-checkAllBtn').onclick = checkAllBindings;
        if ($('ayyb-reloadBindingsBtn')) $('ayyb-reloadBindingsBtn').onclick = () => loadBindings(true);
        if ($('ayyb-scanBtn')) $('ayyb-scanBtn').onclick = startScan;
        if ($('ayyb-qrCloseBtn')) $('ayyb-qrCloseBtn').onclick = closeQr;
        togglePayload();
    }

    function init() {
        bindEvents();
    }

    global.AdminYyb = {
        init,
        loadAll,
        loadDashboard,
        switchTab,
        checkAllBindings,
        loadBindings,
        warmupProtocolCheck,
    };
})(window);
