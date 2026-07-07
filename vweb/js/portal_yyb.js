(function (global) {
    'use strict';

    const API = '/api/portal/yyb';
    const state = { accounts: [], selectedKey: '', lastResult: '', scanSessionId: '', scanTimer: null, scanPolling: false, inited: false };

    function $(id) { return document.getElementById(id); }

    function esc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    async function request(path, options = {}) {
        const res = await fetch(API + path, {
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json', 'X-Request-Source': 'web', ...(options.headers || {}) },
            ...options,
        });
        if (res.redirected && res.url.includes('/portal/login')) {
            location.href = '/portal/login';
            throw new Error('未登录');
        }
        const data = await res.json();
        if (data.code !== 0) throw new Error(data.msg || '请求失败');
        return data.data;
    }

    function accountName(a) { return a.nickname || a.openid || '未命名'; }

    function statusTag(s) {
        if (s === 'alive' || s === 'online') return '<span class="wx-device-badge online-badge">🟢 可用</span>';
        if (s === 'dead' || s === 'offline' || s === 'expired') return '<span class="wx-device-badge offline-badge">🔴 失效</span>';
        return '<span class="wx-device-badge secondary-badge">' + esc(s || '未知') + '</span>';
    }

    function accountKey(acc) {
        if (acc.bindingId) return String(acc.bindingId);
        return String(acc.openid || '').trim();
    }

    function accountRef(acc) { return accountKey(acc); }

    function setResult(val, err) {
        const box = $('yyb-resultBox');
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

    function selectedAccount() { return state.accounts.find(a => accountKey(a) === state.selectedKey); }

    function syncSelected() {
        const label = $('yyb-selectedLabel');
        const a = selectedAccount();
        if (label) label.textContent = a ? ('当前：' + accountName(a)) : '未选择账号（请先点击上方账号卡片）';
        document.querySelectorAll('.yyb-acc-card').forEach(el => {
            el.classList.toggle('selected', el.dataset.key === state.selectedKey);
        });
    }

    function attrEsc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/"/g, '&quot;');
    }

    function formatUin(acc) {
        const u = acc && acc.uin;
        if (u != null && u !== '' && Number(u) > 0) return String(u);
        return '';
    }

    function renderUinLine(acc) {
        const uin = formatUin(acc);
        const display = uin || '未获取';
        const title = uin ? uin : '扫码绑定后会自动获取；若仍为空请点击「刷新账号」';
        return `<div class="yyb-acc-uin-line">
            <span class="yyb-meta-label">UIN</span>
            <code class="yyb-uin-text${uin ? '' : ' missing'}" title="${attrEsc(title)}">${esc(display)}</code>
        </div>`;
    }

    function renderAccountCard(acc) {
        const rawOpenid = String(acc.openid || '');
        const openid = esc(rawOpenid);
        return `<div class="yyb-acc-card" data-key="${attrEsc(accountKey(acc))}" role="button" tabindex="0">
            <div class="yyb-acc-head">
                <div class="yyb-acc-name">${esc(accountName(acc))}</div>
                ${statusTag(acc.status)}
            </div>
            ${renderUinLine(acc)}
            <div class="yyb-acc-openid-line">
                <span class="yyb-meta-label">OpenID</span>
                <code class="yyb-openid-text" title="${attrEsc(rawOpenid)}">${openid || '-'}</code>
                <button type="button" class="yyb-copy-btn" data-yyb-copy="${attrEsc(rawOpenid)}">复制</button>
            </div>
        </div>`;
    }

    function renderAccounts() {
        const grid = $('yyb-accountGrid');
        const countEl = $('yyb-accountCount');
        if (countEl) countEl.textContent = String(state.accounts.length);
        if (!grid) return;
        if (!state.accounts.length) {
            grid.innerHTML = '<div class="yyb-empty">暂无账号，请点击「扫码添加」</div>';
            state.selectedKey = '';
            syncSelected();
            return;
        }
        if (!state.accounts.some(a => accountKey(a) === state.selectedKey)) {
            state.selectedKey = accountKey(state.accounts[0]);
        }
        grid.innerHTML = state.accounts.map(renderAccountCard).join('');
        grid.querySelectorAll('.yyb-acc-card').forEach(card => {
            card.onclick = (e) => {
                if (e.target.closest('.yyb-copy-btn')) return;
                state.selectedKey = card.dataset.key;
                syncSelected();
            };
        });
        syncSelected();
    }

    function renderDashboard(container, st) {
        if (!container) return;
        if (!st || !st.enabled) {
            container.innerHTML = '<div class="yyb-empty">应用宝模块未启用</div>';
            return;
        }
        if (!st.ready) {
            container.innerHTML = '<div class="yyb-empty">' + esc(st.message || '服务暂不可用') + '</div>';
            return;
        }
        const accounts = st.accounts || [];
        if (!accounts.length) {
            container.innerHTML = '<div class="yyb-empty">暂无绑定账号，<a href="#" onclick="document.querySelector(\'button[data-panel=yyb]\')?.click();return false;" style="color:var(--cyan);">去添加</a></div>';
            return;
        }
        const alive = accounts.filter(a => a.status === 'alive' || a.status === 'online').length;
        container.innerHTML = `
            <div class="yyb-chips" style="margin-bottom:10px;">
                <span class="yyb-chip">共 <strong>${accounts.length}</strong> 个</span>
                <span class="yyb-chip ok">可用 <strong>${alive}</strong></span>
                <span class="yyb-chip">扫码 <strong>${st.scanLoginCost ?? '-'}</strong> 积分</span>
            </div>
            <div class="yyb-dash-list">${accounts.slice(0, 4).map(a => `
                <div class="yyb-dash-item">
                    <div class="yyb-dash-item-head">
                        <span class="yyb-dash-item-name">${esc(accountName(a))}</span>
                        ${statusTag(a.status)}
                    </div>
                    <div class="yyb-dash-uin">UIN: ${esc(formatUin(a) || '未获取')}</div>
                    <div class="yyb-dash-openid">OpenID: ${esc(a.openid)}</div>
                </div>`).join('')}</div>
            ${accounts.length > 4 ? '<div style="font-size:11px;color:var(--text-muted);margin-top:8px;text-align:center;">还有 ' + (accounts.length - 4) + ' 个账号…</div>' : ''}`;
    }

    async function loadDashboard() {
        const container = $('yybDashAccounts');
        if (!container) return;
        try {
            const st = await request('/status');
            renderDashboard(container, st);
        } catch (_) {
            container.innerHTML = '<div class="yyb-empty">加载失败</div>';
        }
    }

    const state = { accounts: [], selectedKey: '', lastResult: '', scanSessionId: '', scanTimer: null, scanPolling: false, inited: false, panelLoading: false };

    function setStatusBar(message, type, loading) {
        const bar = $('yyb-statusBar');
        const text = $('yyb-statusText');
        const spin = $('yyb-statusSpin');
        if (!bar || !text) return;
        bar.classList.remove('hidden', 'loading', 'ok', 'warn', 'error');
        if (type) bar.classList.add(type);
        if (loading) bar.classList.add('loading');
        text.textContent = message || '';
        if (spin) spin.style.display = loading ? 'inline-block' : 'none';
    }

    function setHealthStatus(text, chipClass) {
        const chip = $('yyb-healthChip');
        const healthText = $('yyb-healthText');
        if (healthText) healthText.textContent = text || '-';
        if (chip) chip.className = 'yyb-chip' + (chipClass ? (' ' + chipClass) : '');
    }

    function setGridLoading(loading) {
        const grid = $('yyb-accountGrid');
        if (grid) grid.classList.toggle('is-loading', !!loading);
    }

    function setBtnLoading(btn, loading, loadingText) {
        if (!btn) return;
        if (loading) {
            if (!btn.dataset.prevText) btn.dataset.prevText = btn.textContent;
            btn.classList.add('is-loading');
            btn.disabled = true;
            if (loadingText) btn.textContent = loadingText;
        } else {
            btn.classList.remove('is-loading');
            btn.disabled = false;
            if (btn.dataset.prevText) {
                btn.textContent = btn.dataset.prevText;
                delete btn.dataset.prevText;
            }
        }
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

    async function loadPanel(options = {}) {
        if (state.panelLoading) return;
        const autoCheck = !!options.autoCheck;
        const triggerBtn = options.triggerBtn || null;
        state.panelLoading = true;
        setGridLoading(true);
        setBtnLoading(triggerBtn || $('yyb-reloadBtn'), true, autoCheck ? '检测中…' : '刷新中…');
        setHealthStatus(autoCheck ? '检测中…' : '刷新中…', 'checking');
        setStatusBar(autoCheck ? '正在检测全部账号存活状态，请稍候…' : '正在刷新账号列表…', 'loading', true);
        try {
            const st = await request('/status' + (autoCheck ? '?check=1' : ''));
            if ($('yyb-coinVal')) $('yyb-coinVal').textContent = st.coin ?? '-';
            if ($('yyb-scanCostVal')) $('yyb-scanCostVal').textContent = st.scanLoginCost ?? '-';
            if (!st.enabled || !st.ready) {
                setHealthStatus(st.message || '不可用', 'bad');
                if ($('yyb-scanBtn')) $('yyb-scanBtn').disabled = true;
                setStatusBar(st.message || '应用宝服务暂不可用', 'error', false);
            } else {
                setHealthStatus(autoCheck ? '检测完成' : '已刷新', 'ok');
                if ($('yyb-scanBtn')) $('yyb-scanBtn').disabled = false;
                if (autoCheck && st.checkSummary) {
                    const summaryText = formatCheckSummary(st.checkSummary);
                    const dead = Number(st.checkSummary.dead || 0);
                    const failed = Number(st.checkSummary.failed || 0);
                    setStatusBar(summaryText, (dead > 0 || failed > 0) ? 'warn' : 'ok', false);
                    if (typeof global.toast === 'function') {
                        global.toast(summaryText, dead > 0 ? 'error' : 'success');
                    }
                } else {
                    setStatusBar('账号列表已刷新', 'ok', false);
                    if (!autoCheck && typeof global.toast === 'function') {
                        global.toast('刷新完成', 'success');
                    }
                }
            }
            state.accounts = st.accounts || [];
            renderAccounts();
            loadDashboard();
        } catch (e) {
            setHealthStatus('加载失败', 'bad');
            setStatusBar(e.message || '加载失败，请稍后重试', 'error', false);
            if (typeof global.toast === 'function') global.toast(e.message || '加载失败', 'error');
        } finally {
            state.panelLoading = false;
            setGridLoading(false);
            setBtnLoading(triggerBtn || $('yyb-reloadBtn'), false);
        }
    }

    async function withAccountAction(btn, loadingLabel, resultHint, action) {
        if (!btn || btn.disabled) return;
        const prev = btn.textContent;
        const peer = [$('yyb-refreshBtn'), $('yyb-resyncBtn')].filter(b => b && b !== btn);
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
        const g = $('yyb-payloadGroup');
        const sel = $('yyb-featureSel');
        if (g && sel) g.classList.toggle('hidden', sel.value.toLowerCase() !== 'operatewxdata');
    }

    async function callFeature() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择一个账号', true); return; }
        const feature = $('yyb-featureSel').value;
        const appId = $('yyb-appidInput').value.trim();
        if (!appId) { setResult('请输入 AppID', true); return; }
        const body = { ref: accountRef(acc), appId };
        let path = '/wxapp/getCode';
        if (feature === 'getPhoneNumber') path = '/wxapp/getPhoneNumber';
        if (feature === 'operateWxData') {
            path = '/wxapp/operateWxData';
            try { body.payload = JSON.parse($('yyb-payloadInput').value || '{}'); }
            catch { setResult('JSON 格式错误', true); return; }
        }
        $('yyb-callBtn').disabled = true;
        setResult('调用中…', false);
        try {
            setResult(await request(path, { method: 'POST', body: JSON.stringify(body) }), false);
            await loadPanel({ autoCheck: false });
        } catch (e) { setResult(e.message, true); }
        finally { $('yyb-callBtn').disabled = false; }
    }

    async function refreshSelected() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择一个账号', true); return; }
        try {
            await withAccountAction($('yyb-refreshBtn'), '刷新中…', '正在刷新存活状态，请稍候…', async () => {
                setStatusBar('正在刷新选中账号存活状态…', 'loading', true);
                setResult(await request('/accounts/refresh', { method: 'POST', body: JSON.stringify({ ref: accountRef(acc) }) }), false);
                await loadPanel({ autoCheck: false, triggerBtn: $('yyb-refreshBtn') });
                if (typeof global.toast === 'function') global.toast('账号存活状态已刷新', 'success');
            });
        } catch (e) { setResult(e.message, true); }
    }

    async function resyncSelected() {
        const acc = selectedAccount();
        if (!acc) { setResult('请先选择一个账号', true); return; }
        try {
            await withAccountAction($('yyb-resyncBtn'), '同步中…', '正在同步账号资料，请稍候…', async () => {
                setResult(await request('/accounts/resync', { method: 'POST', body: JSON.stringify({ ref: accountRef(acc) }) }), false);
                await loadPanel({ autoCheck: true });
            });
        } catch (e) { setResult(e.message, true); }
    }

    async function deleteSelected() {
        const acc = selectedAccount();
        if (!acc || !confirm('确定删除该账号？')) return;
        try {
            await request('/accounts/delete', { method: 'POST', body: JSON.stringify({ ref: accountRef(acc) }) });
            await loadPanel({ autoCheck: false });
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
        const m = $('yyb-qrModal');
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
            if (status === 'scanned') {
                $('yyb-qrHint').textContent = '已扫码，请在手机上点击「确认登录」';
                keepPolling = true;
                return;
            }
            if (status === 'authorized' || status === 'confirmed') {
                stopScanPoll();
                $('yyb-qrHint').textContent = '正在完成绑定（获取 UIN 中，约需数秒）…';
                const result = await request('/qr/' + encodeURIComponent(state.scanSessionId) + '/confirm', { method: 'POST' });
                closeQr();
                if (result.alreadyBound) {
                    setResult('扫码成功，账号已绑定。', false);
                } else if (result.cost > 0) {
                    setResult('扫码登录成功，已扣除 ' + result.cost + ' 积分。', false);
                } else {
                    setResult('扫码登录成功，账号已绑定。', false);
                }
                await loadPanel({ autoCheck: true });
                return;
            }
            if (['expired', 'cancelled', 'unknown'].includes(status)) {
                stopScanPoll();
                $('yyb-qrHint').textContent = status === 'expired' ? '二维码已过期，请重新生成' : '扫码已取消';
                return;
            }
            keepPolling = true;
        } catch (e) {
            const msg = e.message || '';
            if (/deadline|timeout/i.test(msg)) {
                $('yyb-qrHint').textContent = '等待扫码中（网络较慢）…';
                keepPolling = true;
                return;
            }
            stopScanPoll();
            $('yyb-qrHint').textContent = msg || '登录失败';
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
            $('yyb-qrBox').innerHTML = src ? '<img src="' + src + '" alt="二维码" style="width:180px;height:180px;">' : '加载失败';
            const cost = data.scanLoginCost ?? '-';
            $('yyb-qrHint').textContent = data.scanCostHint || '请使用微信扫码确认登录';
            $('yyb-qrModal').classList.add('show');
            stopScanPoll();
            scheduleScanPoll(0);
        } catch (e) { setResult(e.message, true); }
    }

    function bindEvents() {
        if (state.inited) return;
        state.inited = true;
        setupCopy();
        if ($('yyb-featureSel')) $('yyb-featureSel').onchange = togglePayload;
        if ($('yyb-callBtn')) $('yyb-callBtn').onclick = callFeature;
        if ($('yyb-clearBtn')) $('yyb-clearBtn').onclick = () => setResult('结果已清空', false);
        if ($('yyb-copyBtn')) $('yyb-copyBtn').onclick = () => {
            const text = state.lastResult || ($('yyb-resultBox') && $('yyb-resultBox').textContent) || '';
            copyText(text, $('yyb-copyBtn'));
        };
        if ($('yyb-reloadBtn')) $('yyb-reloadBtn').onclick = () => loadPanel({ autoCheck: true, triggerBtn: $('yyb-reloadBtn') });
        if ($('yyb-refreshBtn')) $('yyb-refreshBtn').onclick = refreshSelected;
        if ($('yyb-resyncBtn')) $('yyb-resyncBtn').onclick = resyncSelected;
        if ($('yyb-deleteBtn')) $('yyb-deleteBtn').onclick = deleteSelected;
        if ($('yyb-scanBtn')) $('yyb-scanBtn').onclick = startScan;
        if ($('yyb-qrCloseBtn')) $('yyb-qrCloseBtn').onclick = closeQr;
        togglePayload();
    }

    function init() {
        bindEvents();
    }

    global.YybPortal = {
        init,
        loadPanel,
        loadDashboard,
    };
})(window);
