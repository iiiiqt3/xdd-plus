(function (global) {
    'use strict';

    const API = '/api/portal/yyb';
    const PROTOCOL_API = '/api/portal/protocol';
    const state = { accounts: [], selectedKey: '', scanSessionId: '', scanTimer: null, scanPolling: false, inited: false, panelLoading: false, proxyEnabled: false, proxyPackId: '', proxyBypassRegionName: '', proxyAreasLoaded: false, sessionAliveChecked: false, expiryTimer: null };
    const YYB_LOGIN_VALID_MS = 30 * 24 * 60 * 60 * 1000;

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

    async function protocolRequest(path, options = {}) {
        const res = await fetch(PROTOCOL_API + path, {
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

    function areaRows(data) {
        if (!data) return [];
        if (Array.isArray(data.list)) return data.list;
        if (Array.isArray(data.provinceList)) return data.provinceList;
        if (Array.isArray(data.city)) return data.city;
        if (Array.isArray(data.cityList)) return data.cityList;
        return [];
    }

    function fillSelect(sel, rows, placeholder) {
        if (!sel) return;
        sel.innerHTML = '';
        const ph = document.createElement('option');
        ph.value = '';
        ph.textContent = placeholder || '请选择';
        sel.appendChild(ph);
        rows.forEach(function (row) {
            const opt = document.createElement('option');
            opt.value = String(row.regionCode || row.region_code || '');
            opt.textContent = row.regionName || row.region_name || opt.value;
            sel.appendChild(opt);
        });
    }

    async function loadProxyConfig() {
        try {
            const cfg = await protocolRequest('/proxy/config');
            state.proxyEnabled = !!cfg.proxyEnabled;
            state.proxyPackId = cfg.proxyDefaultPackid || '';
            state.proxyBypassRegionName = cfg.proxyBypassRegionName || '';
            const hint = $('yyb-proxyHint');
            const warn = $('yyb-proxyConfigWarn');
            if (hint) {
                if (state.proxyEnabled && state.proxyBypassRegionName) {
                    hint.textContent = '「' + state.proxyBypassRegionName + '」等地区免代理，将直连登录；其他地区将自动匹配当地 IP。';
                } else if (state.proxyEnabled) {
                    hint.textContent = '请选择与你当前所在地一致的省/市，系统将自动匹配当地 IP 登录。';
                }
            }
            if (warn) {
                const ok = cfg.proxyAccountConfigured !== false;
                warn.style.display = state.proxyEnabled && !ok ? 'block' : 'none';
            }
        } catch (e) {
            console.warn('loadProxyConfig', e);
        }
    }

    async function ensureProxyAreasLoaded() {
        if (!state.proxyEnabled || state.proxyAreasLoaded) return;
        await loadProxyProvinces();
        state.proxyAreasLoaded = true;
    }

    async function loadProxyProvinces() {
        const data = await protocolRequest('/proxy/areas', {
            method: 'POST',
            body: JSON.stringify({ packid: state.proxyPackId || '' }),
        });
        fillSelect($('yyb-proxyProvince'), areaRows(data), '请选择省份');
        fillSelect($('yyb-proxyCity'), [], '请先选择省份');
    }

    async function loadProxyCities() {
        const prov = $('yyb-proxyProvince');
        const code = prov ? prov.value : '';
        if (!code) {
            fillSelect($('yyb-proxyCity'), [], '请先选择省份');
            return;
        }
        const data = await protocolRequest('/proxy/areas', {
            method: 'POST',
            body: JSON.stringify({ parent_code: code, packid: state.proxyPackId || '' }),
        });
        fillSelect($('yyb-proxyCity'), areaRows(data), '请选择城市');
    }

    function accountRef(acc) { return accountKey(acc); }

    function notify(msg, type) {
        if (typeof global.toast === 'function') global.toast(msg, type || 'info');
    }

    function selectedProxyRegion() {
        const citySel = $('yyb-proxyCity');
        if (!citySel || citySel.selectedIndex <= 0) return { regionCode: '', regionName: '' };
        const opt = citySel.options[citySel.selectedIndex];
        return { regionCode: opt.value || '', regionName: (opt.textContent || '').trim() };
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

    function renderExpiryLine(acc) {
        const loginSec = Number(acc.loginAt || acc.createdAt || 0);
        if (!loginSec) return '';
        const expireMs = (Number(acc.expiresAt) || (loginSec + YYB_LOGIN_VALID_MS / 1000)) * 1000;
        const remain = expireMs - Date.now();
        if (remain <= 0) {
            return '<div class="yyb-acc-expiry expired">登录已过期，请重新扫码登录延期</div>';
        }
        const days = Math.floor(remain / 86400000);
        const hours = Math.floor((remain % 86400000) / 3600000);
        const cls = remain < 3 * 86400000 ? ' warn' : '';
        const expireText = new Date(expireMs).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
        return '<div class="yyb-acc-expiry' + cls + '">剩余有效期 <strong>' + days + '</strong> 天 <strong>' + hours + '</strong> 小时<span style="opacity:.75;">（至 ' + esc(expireText) + '）</span></div>';
    }

    function updateExpiryLines() {
        state.accounts.forEach(function (acc) {
            const key = accountKey(acc);
            const el = document.querySelector('.yyb-acc-card[data-key="' + CSS.escape(key) + '"] .yyb-acc-expiry-slot');
            if (el) el.innerHTML = renderExpiryLine(acc);
        });
    }

    function renderAccountCard(acc) {
        const rawOpenid = String(acc.openid || '');
        const openid = esc(rawOpenid);
        const isBound = global.PortalProtocolBind && global.PortalProtocolBind.hasYybBinding(rawOpenid);
        const boundBlock = global.PortalProtocolBind ? global.PortalProtocolBind.renderYybBoundBlock(rawOpenid) : '';
        return `<div class="yyb-acc-card${isBound ? ' proto-bound' : ''}" data-key="${attrEsc(accountKey(acc))}" role="button" tabindex="0">
            <div class="yyb-acc-head">
                <div class="yyb-acc-name-wrap">
                    <div class="yyb-acc-name">${esc(accountName(acc))}</div>
                </div>
                ${statusTag(acc.status)}
            </div>
            ${renderUinLine(acc)}
            <div class="yyb-acc-openid-line">
                <span class="yyb-meta-label">OpenID</span>
                <code class="yyb-openid-text" title="${attrEsc(rawOpenid)}">${openid || '-'}</code>
                <button type="button" class="yyb-copy-btn" data-yyb-copy="${attrEsc(rawOpenid)}">复制</button>
            </div>
            ${boundBlock}
            <div class="yyb-acc-expiry-slot">${renderExpiryLine(acc)}</div>
        </div>`;
    }

    function startExpiryTicker() {
        if (state.expiryTimer) clearInterval(state.expiryTimer);
        state.expiryTimer = setInterval(updateExpiryLines, 60000);
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
        startExpiryTicker();
        grid.querySelectorAll('.yyb-acc-card').forEach(card => {
            card.onclick = (e) => {
                if (e.target.closest('.yyb-copy-btn') || e.target.closest('.proto-card-unbind-btn')) return;
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
            container.innerHTML = '<div class="yyb-empty">暂无绑定账号，<a href="#" onclick="openProtocolPanel(\'yyb\');return false;" style="color:var(--cyan);">去添加</a></div>';
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
        const autoCheck = !!options.autoCheck;
        const silent = !!options.silent;
        const triggerBtn = options.triggerBtn || null;
        if (state.panelLoading) {
            const busyMsg = autoCheck ? '正在检测中，请稍候…' : '正在刷新中，请稍候…';
            setStatusBar(busyMsg, 'loading', true);
            if (!silent && typeof global.toast === 'function') global.toast(busyMsg, 'info');
            return;
        }
        state.panelLoading = true;
        if (autoCheck && !silent && typeof global.toast === 'function') {
            global.toast('开始检测应用宝账号状态…', 'info');
        }
        setGridLoading(true);
        setBtnLoading(triggerBtn || $('yyb-reloadBtn'), true, autoCheck ? '检测中…' : '刷新中…');
        if (autoCheck) {
            setHealthStatus(silent ? '检测中…' : '检测中…', 'checking');
            setStatusBar(silent ? '正在后台检测账号状态…' : '正在检测全部账号存活状态，请稍候…', 'loading', true);
        } else {
            setHealthStatus('刷新中…', 'checking');
            setStatusBar('正在刷新账号列表…', 'loading', true);
        }
        try {
            const st = await request('/status' + (autoCheck ? '?check=1' : ''));
            if ($('yyb-coinVal')) $('yyb-coinVal').textContent = st.coin ?? '-';
            if ($('yyb-scanCostVal')) $('yyb-scanCostVal').textContent = st.scanLoginCost ?? '-';
            if (!st.enabled || !st.ready) {
                setHealthStatus(st.message || '不可用', 'bad');
                if ($('yyb-scanBtn')) $('yyb-scanBtn').disabled = true;
                setStatusBar(st.message || '应用宝服务暂不可用', 'error', false);
            } else {
                if ($('yyb-scanBtn')) $('yyb-scanBtn').disabled = false;
                if (autoCheck) {
                    const summaryText = st.checkSummary ? formatCheckSummary(st.checkSummary) : '检测完成';
                    const dead = Number((st.checkSummary && st.checkSummary.dead) || 0);
                    const failed = Number((st.checkSummary && st.checkSummary.failed) || 0);
                    setHealthStatus('检测完成', 'ok');
                    setStatusBar(summaryText, (dead > 0 || failed > 0) ? 'warn' : 'ok', false);
                    if (!silent && typeof global.toast === 'function') {
                        const toastType = dead > 0 ? 'error' : (failed > 0 ? 'warn' : 'success');
                        global.toast(summaryText, toastType);
                    }
                } else {
                    setHealthStatus('已刷新', 'ok');
                    setStatusBar('账号列表已刷新', 'ok', false);
                    if (!silent && typeof global.toast === 'function') {
                        global.toast('刷新完成', 'success');
                    }
                }
            }
            state.accounts = st.accounts || [];
            if (global.PortalProtocolBind) {
                global.PortalProtocolBind.setYybAccounts(state.accounts);
                void global.PortalProtocolBind.refresh();
            }
            renderAccounts();
            loadDashboard();
            void loadProxyConfig();
        } catch (e) {
            setHealthStatus('加载失败', 'bad');
            setStatusBar(e.message || '加载失败，请稍后重试', 'error', false);
            if (!silent && typeof global.toast === 'function') global.toast(e.message || '加载失败', 'error');
        } finally {
            state.panelLoading = false;
            setGridLoading(false);
            setBtnLoading(triggerBtn || $('yyb-reloadBtn'), false);
        }
    }

    async function runInitialSessionCheck() {
        if (state.sessionAliveChecked) return;
        state.sessionAliveChecked = true;
        await loadPanel({ autoCheck: true, silent: true });
    }

    async function withAccountAction(btn, loadingLabel, action) {
        if (!btn || btn.disabled) return;
        const prev = btn.textContent;
        const peer = [$('yyb-refreshBtn')].filter(b => b && b !== btn);
        btn.disabled = true;
        peer.forEach(b => { b.disabled = true; });
        btn.textContent = loadingLabel;
        try { await action(); } finally {
            btn.disabled = false;
            btn.textContent = prev;
            peer.forEach(b => { b.disabled = false; });
        }
    }

    async function refreshSelected() {
        const acc = selectedAccount();
        if (!acc) { notify('请先点击选择一个账号', 'warning'); return; }
        try {
            await withAccountAction($('yyb-refreshBtn'), '刷新中…', async () => {
                setStatusBar('正在刷新选中账号存活状态…', 'loading', true);
                await request('/accounts/refresh', { method: 'POST', body: JSON.stringify({ ref: accountRef(acc) }) });
                await loadPanel({ autoCheck: false, silent: true, triggerBtn: $('yyb-refreshBtn') });
                notify('账号存活状态已刷新', 'success');
            });
        } catch (e) { notify(e.message || '刷新失败', 'error'); }
    }

    async function deleteSelected() {
        const acc = selectedAccount();
        if (!acc || !confirm('确定删除该账号？')) return;
        try {
            await request('/accounts/delete', { method: 'POST', body: JSON.stringify({ ref: accountRef(acc) }) });
            notify('账号已删除', 'success');
            await loadPanel({ autoCheck: false, silent: true });
        } catch (e) { notify(e.message || '删除失败', 'error'); }
    }

    function stopScanPoll() {
        if (state.scanTimer) { clearTimeout(state.scanTimer); state.scanTimer = null; }
        state.scanPolling = false;
    }

    function scheduleScanPoll(delay) {
        if (state.scanTimer) clearTimeout(state.scanTimer);
        state.scanTimer = setTimeout(() => { void pollScan(); }, delay);
    }

    function resetQrModal() {
        state.scanSessionId = '';
        const regionStep = $('yyb-scanRegionStep');
        const qrStep = $('yyb-scanQrStep');
        const title = $('yyb-qrModalTitle');
        if (regionStep) regionStep.style.display = '';
        if (qrStep) qrStep.style.display = 'none';
        if (title) title.textContent = '扫码添加账号';
        const regionTag = $('yyb-qrRegionTag');
        if (regionTag) {
            regionTag.style.display = 'none';
            regionTag.textContent = '';
        }
        const costEl = $('yyb-qrCost');
        if (costEl) {
            costEl.style.display = 'none';
            costEl.textContent = '';
        }
        const box = $('yyb-qrBox');
        if (box) box.innerHTML = '等待生成…';
        const hint = $('yyb-qrHint');
        if (hint) hint.textContent = '请使用微信扫码确认登录';
        const prov = $('yyb-proxyProvince');
        const city = $('yyb-proxyCity');
        if (prov) prov.value = '';
        if (city) fillSelect(city, [], '请先选择省份');
        setBtnLoading($('yyb-regionConfirmBtn'), false);
    }

    function showRegionStep() {
        const regionStep = $('yyb-scanRegionStep');
        const qrStep = $('yyb-scanQrStep');
        const title = $('yyb-qrModalTitle');
        if (regionStep) regionStep.style.display = '';
        if (qrStep) qrStep.style.display = 'none';
        if (title) title.textContent = '选择登录地区';
    }

    function showQrStep(region) {
        const regionStep = $('yyb-scanRegionStep');
        const qrStep = $('yyb-scanQrStep');
        const title = $('yyb-qrModalTitle');
        if (regionStep) regionStep.style.display = 'none';
        if (qrStep) qrStep.style.display = '';
        if (title) title.textContent = '微信扫码';
        const regionTag = $('yyb-qrRegionTag');
        if (regionTag) {
            if (region && region.regionName) {
                regionTag.style.display = 'block';
                regionTag.textContent = '登录地区：' + region.regionName;
            } else {
                regionTag.style.display = 'none';
                regionTag.textContent = '';
            }
        }
    }

    function closeQr() {
        const m = $('yyb-qrModal');
        if (m) m.classList.remove('show');
        stopScanPoll();
        resetQrModal();
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
                    notify('扫码成功，账号已绑定。', 'success');
                } else if (result.cost > 0) {
                    notify('扫码登录成功，已扣除 ' + result.cost + ' 积分。', 'success');
                } else {
                    notify('扫码登录成功，账号已绑定。', 'success');
                }
                await loadPanel({ autoCheck: false, silent: true });
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
            notify(msg || '扫码确认失败', 'error');
        } finally {
            state.scanPolling = false;
            if (keepPolling && state.scanSessionId) scheduleScanPoll(800);
        }
    }

    async function generateQr(region) {
        const scanBtn = $('yyb-scanBtn');
        setBtnLoading(scanBtn, true, '生成中…');
        setBtnLoading($('yyb-regionConfirmBtn'), true, '生成中…');
        $('yyb-qrBox').innerHTML = '<span class="muted">正在' + (state.proxyEnabled && region.regionCode ? '提取「' + esc(region.regionName) + '」代理并' : '') + '生成二维码…</span>';
        $('yyb-qrHint').textContent = '请稍候';
        try {
            const payload = {
                useProxy: state.proxyEnabled,
                regionCode: region.regionCode,
                regionName: region.regionName,
                packId: state.proxyPackId || '',
            };
            const data = await request('/qr', {
                method: 'POST',
                body: JSON.stringify(payload),
            });
            state.scanSessionId = data.sessionId;
            const src = qrImgSrc(data.imageBase64);
            $('yyb-qrBox').innerHTML = src ? '<img src="' + src + '" alt="二维码" style="width:180px;height:180px;">' : '加载失败';
            $('yyb-qrHint').textContent = '请使用微信扫码确认登录';
            if (data.proxyBypass) {
                const regionTag = $('yyb-qrRegionTag');
                if (regionTag && region.regionName) {
                    regionTag.style.display = 'block';
                    regionTag.textContent = '登录地区：' + region.regionName + '（免代理直连）';
                }
            }
            const costEl = $('yyb-qrCost');
            const cost = Number(data.scanLoginCost);
            if (costEl) {
                if (Number.isFinite(cost) && cost > 0) {
                    costEl.style.display = 'block';
                    costEl.textContent = '本次扫码将扣除 ' + cost + ' 积分（确认登录后扣除）';
                } else if (Number.isFinite(cost) && cost === 0) {
                    costEl.style.display = 'block';
                    costEl.textContent = '本次扫码免费，不扣除积分';
                } else if (data.scanCostHint) {
                    costEl.style.display = 'block';
                    costEl.textContent = data.scanCostHint;
                } else {
                    costEl.style.display = 'none';
                    costEl.textContent = '';
                }
            }
            stopScanPoll();
            scheduleScanPoll(0);
        } catch (e) {
            $('yyb-qrBox').innerHTML = '<span style="color:#f56c6c;font-size:13px;line-height:1.6;">' + esc(e.message || '生成失败') + '</span>';
            $('yyb-qrHint').textContent = '生成失败，请查看提示或换地区重试';
            if (typeof global.toast === 'function') global.toast(e.message || '生成失败', 'error');
        } finally {
            setBtnLoading(scanBtn, false);
            setBtnLoading($('yyb-regionConfirmBtn'), false);
        }
    }

    async function confirmRegionAndScan() {
        const region = selectedProxyRegion();
        if (state.proxyEnabled && !region.regionCode) {
            notify('请选择你当前所在的省/市', 'warning');
            return;
        }
        showQrStep(region);
        await generateQr(region);
    }

    async function startScan() {
        stopScanPoll();
        resetQrModal();
        $('yyb-qrModal').classList.add('show');
        if (state.proxyEnabled) {
            showRegionStep();
            try {
                await ensureProxyAreasLoaded();
            } catch (e) {
                notify(e.message || '加载地区列表失败', 'error');
            }
            return;
        }
        showQrStep({ regionCode: '', regionName: '' });
        await generateQr({ regionCode: '', regionName: '' });
    }

    function bindEvents() {
        if (state.inited) return;
        state.inited = true;
        setupCopy();
        if ($('yyb-reloadBtn')) $('yyb-reloadBtn').onclick = () => {
            loadPanel({ autoCheck: true, triggerBtn: $('yyb-reloadBtn') });
        };
        if ($('yyb-refreshBtn')) $('yyb-refreshBtn').onclick = refreshSelected;
        if ($('yyb-deleteBtn')) $('yyb-deleteBtn').onclick = deleteSelected;
        if ($('yyb-scanBtn')) $('yyb-scanBtn').onclick = startScan;
        if ($('yyb-regionConfirmBtn')) $('yyb-regionConfirmBtn').onclick = () => { void confirmRegionAndScan(); };
        if ($('yyb-regionCancelBtn')) $('yyb-regionCancelBtn').onclick = closeQr;
        if ($('yyb-proxyProvince')) $('yyb-proxyProvince').onchange = () => { void loadProxyCities(); };
        if ($('yyb-qrCloseBtn')) $('yyb-qrCloseBtn').onclick = closeQr;
        void loadProxyConfig();
    }

    function init() {
        bindEvents();
    }

    global.YybPortal = {
        init,
        loadPanel,
        loadDashboard,
        runInitialSessionCheck,
    };
})(window);
