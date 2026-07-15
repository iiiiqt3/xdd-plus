(function (global) {
    'use strict';

    const state = {
        bindings: [],
        wxDevices: [],
        yybAccounts: [],
        quota: null,
        loading: false,
    };

    function esc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    function attrEsc(s) {
        return String(s ?? '').replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    }

    function shorten(id, head, tail) {
        id = String(id || '').trim();
        head = head || 8;
        tail = tail || 6;
        if (id.length <= head + tail + 3) return id;
        return id.slice(0, head) + '…' + id.slice(-tail);
    }

    function request(path, options) {
        if (typeof global.request === 'function') {
            return global.request(path, options);
        }
        return fetch(path, {
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json', 'X-Request-Source': 'web', ...(options && options.headers) },
            ...options,
        }).then(function (res) {
            return res.json().then(function (data) {
                if (data.code !== 0) throw new Error(data.msg || '请求失败');
                return data;
            });
        });
    }

    function toast(msg, type) {
        if (typeof global.toast === 'function') global.toast(msg, type);
    }

    function bindingByWx(wxid) {
        wxid = String(wxid || '').trim();
        return state.bindings.find(function (b) { return b.wxWxid === wxid; }) || null;
    }

    function bindingByOpenID(openid) {
        openid = String(openid || '').trim();
        return state.bindings.find(function (b) { return b.yybOpenId === openid; }) || null;
    }

    function boundWxSet() {
        return new Set(state.bindings.map(function (b) { return b.wxWxid; }));
    }

    function boundOpenIDSet() {
        return new Set(state.bindings.map(function (b) { return b.yybOpenId; }));
    }

    function unboundWxDevices() {
        var bound = boundWxSet();
        return (state.wxDevices || []).filter(function (d) {
            return d && d.wxid && !bound.has(d.wxid);
        });
    }

    function unboundYybAccounts() {
        var bound = boundOpenIDSet();
        return (state.yybAccounts || []).filter(function (a) {
            return a && a.openid && !bound.has(a.openid);
        });
    }

    function shouldShowBindUI() {
        return (state.wxDevices || []).length > 0 && (state.yybAccounts || []).length > 0;
    }

    function yybAccountByOpenID(openid) {
        openid = String(openid || '').trim();
        return (state.yybAccounts || []).find(function (a) { return a.openid === openid; }) || null;
    }

    function wxDeviceByWxid(wxid) {
        wxid = String(wxid || '').trim();
        return (state.wxDevices || []).find(function (d) { return d.wxid === wxid; }) || null;
    }

    function updateVisibility() {
        var show = shouldShowBindUI();
        document.querySelectorAll('.proto-bind-card').forEach(function (el) {
            el.classList.toggle('hidden', !show);
        });
    }

    function fillSelect(sel, items, placeholder) {
        if (!sel) return;
        var current = sel.value;
        sel.innerHTML = '';
        var ph = document.createElement('option');
        ph.value = '';
        ph.textContent = placeholder || '请选择';
        sel.appendChild(ph);
        items.forEach(function (item) {
            var opt = document.createElement('option');
            opt.value = item.value;
            opt.textContent = item.label;
            sel.appendChild(opt);
        });
        if (current && Array.from(sel.options).some(function (o) { return o.value === current; })) {
            sel.value = current;
        }
    }

    function syncBindDropdowns() {
        var wxItems = unboundWxDevices().map(function (d) {
            var label = (d.nickname || '微信设备') + ' · ' + shorten(d.wxid);
            if (!d.online) label += '（离线）';
            return { value: d.wxid, label: label };
        });
        var yybItems = unboundYybAccounts().map(function (a) {
            var alive = a.status === 'alive' || a.status === 'online';
            var label = (a.nickname || '应用宝账号') + ' · ' + shorten(a.openid);
            if (!alive) label += '（失效）';
            return { value: a.openid, label: label };
        });
        document.querySelectorAll('[data-proto-bind-wx]').forEach(function (sel) {
            fillSelect(sel, wxItems, '选择微信 wxid');
        });
        document.querySelectorAll('[data-proto-bind-yyb]').forEach(function (sel) {
            fillSelect(sel, yybItems, '选择应用宝 openid');
        });
        document.querySelectorAll('[data-proto-bind-submit]').forEach(function (btn) {
            var card = btn.closest('.proto-bind-card');
            var wxSel = card && card.querySelector('[data-proto-bind-wx]');
            var yybSel = card && card.querySelector('[data-proto-bind-yyb]');
            var disabled = !wxItems.length || !yybItems.length;
            btn.disabled = disabled;
            btn.title = disabled ? '暂无可绑定的微信或应用宝账号' : '';
        });
    }

    function renderBindPairList() {
        document.querySelectorAll('[data-proto-bind-list]').forEach(function (listEl) {
            if (!listEl) return;
            if (!state.bindings.length) {
                listEl.innerHTML = '<div class="proto-bind-empty">暂无双绑记录，请在上方下拉框选择后绑定</div>';
                return;
            }
            listEl.innerHTML = state.bindings.map(function (r) {
                var wx = wxDeviceByWxid(r.wxWxid);
                var yyb = yybAccountByOpenID(r.yybOpenId);
                var wxName = esc((wx && wx.nickname) || r.nickname || '微信设备');
                var yybName = esc((yyb && yyb.nickname) || '应用宝账号');
                var wxid = esc(r.wxWxid || '');
                var oid = esc(r.yybOpenId || '');
                return '<div class="proto-bind-pair-card">' +
                    '<div class="proto-bind-side wx">' +
                    '<span class="proto-bind-side-tag">微信</span>' +
                    '<strong class="proto-bind-side-name">' + wxName + '</strong>' +
                    '<code class="proto-bind-side-id">' + wxid + '</code>' +
                    '</div>' +
                    '<div class="proto-bind-connector" aria-hidden="true"><span>↔</span></div>' +
                    '<div class="proto-bind-side yyb">' +
                    '<span class="proto-bind-side-tag">应用宝</span>' +
                    '<strong class="proto-bind-side-name">' + yybName + '</strong>' +
                    '<code class="proto-bind-side-id">' + oid + '</code>' +
                    '</div>' +
                    '<button class="btn small secondary proto-bind-unbind-btn" type="button" data-wx="' + attrEsc(r.wxWxid) + '" data-oid="' + attrEsc(r.yybOpenId) + '">解绑</button>' +
                    '</div>';
            }).join('');
        });
    }

    function renderQuota() {
        var q = state.quota || {};
        var text = '在线微信 ' + (q.onlineWxSlots || 0) + ' · 已绑 ' + (q.boundPairs || 0) + ' · 免费名额 ' + (q.freeSlots || 0);
        document.querySelectorAll('[data-proto-bind-quota]').forEach(function (el) {
            el.textContent = text;
        });
    }

    function renderWxPeerStrip(wxid) {
        var binding = bindingByWx(wxid);
        if (!binding) return '';
        var yyb = yybAccountByOpenID(binding.yybOpenId);
        var name = esc((yyb && yyb.nickname) || binding.nickname || '应用宝账号');
        var oid = esc(binding.yybOpenId || '');
        return '<div class="proto-peer-strip proto-peer-on-wx">' +
            '<div class="proto-peer-strip-head">' +
            '<span class="proto-peer-strip-badge">🔗 已绑定应用宝</span>' +
            '<button type="button" class="proto-peer-unbind-btn" data-wx="' + attrEsc(binding.wxWxid) + '" data-oid="' + attrEsc(binding.yybOpenId) + '">解绑</button>' +
            '</div>' +
            '<div class="proto-peer-strip-body">' +
            '<strong>' + name + '</strong>' +
            '<code title="' + oid + '">' + esc(shorten(binding.yybOpenId, 10, 8)) + '</code>' +
            '</div>' +
            '</div>';
    }

    function renderYybPeerStrip(openid) {
        var binding = bindingByOpenID(openid);
        if (!binding) return '';
        var wx = wxDeviceByWxid(binding.wxWxid);
        var name = esc((wx && wx.nickname) || binding.nickname || '微信设备');
        var wxid = esc(binding.wxWxid || '');
        return '<div class="proto-peer-strip proto-peer-on-yyb">' +
            '<div class="proto-peer-strip-head">' +
            '<span class="proto-peer-strip-badge">🔗 已绑定微信</span>' +
            '<button type="button" class="proto-peer-unbind-btn" data-wx="' + attrEsc(binding.wxWxid) + '" data-oid="' + attrEsc(binding.yybOpenId) + '">解绑</button>' +
            '</div>' +
            '<div class="proto-peer-strip-body">' +
            '<strong>' + name + '</strong>' +
            '<code title="' + wxid + '">' + esc(shorten(binding.wxWxid, 10, 8)) + '</code>' +
            '</div>' +
            '</div>';
    }

    function hasWxBinding(wxid) {
        return !!bindingByWx(wxid);
    }

    function hasYybBinding(openid) {
        return !!bindingByOpenID(openid);
    }

    async function ensureWxDevices() {
        if ((state.wxDevices || []).length) return;
        try {
            var resp = await request('/api/portal/wx/devices');
            state.wxDevices = resp.data || [];
            if (global.state) global.state.wxDevices = state.wxDevices;
        } catch (_) {}
    }

    async function ensureYybAccounts() {
        if ((state.yybAccounts || []).length) return;
        try {
            var resp = await request('/api/portal/yyb/status');
            state.yybAccounts = (resp.data && resp.data.accounts) || [];
        } catch (_) {}
    }

    async function reloadBindings() {
        if (state.loading) return;
        state.loading = true;
        try {
            await Promise.all([ensureWxDevices(), ensureYybAccounts()]);
            var results = await Promise.all([
                request('/api/portal/protocol/bind/quota'),
                request('/api/portal/protocol/bindings'),
            ]);
            state.quota = results[0].data || {};
            state.bindings = results[1].data || [];
            renderQuota();
            renderBindPairList();
            syncBindDropdowns();
            updateVisibility();
        } catch (e) {
            document.querySelectorAll('[data-proto-bind-list]').forEach(function (el) {
                el.textContent = e.message || '加载失败';
            });
        } finally {
            state.loading = false;
        }
    }

    async function bindFromCard(card) {
        if (!card) return;
        var wxSel = card.querySelector('[data-proto-bind-wx]');
        var yybSel = card.querySelector('[data-proto-bind-yyb]');
        var wx = wxSel ? wxSel.value.trim() : '';
        var oid = yybSel ? yybSel.value.trim() : '';
        if (!wx || !oid) {
            toast('请从下拉框选择微信和应用宝账号', 'warning');
            return;
        }
        var yyb = yybAccountByOpenID(oid);
        try {
            await request('/api/portal/protocol/bind', {
                method: 'POST',
                body: JSON.stringify({
                    wxWxid: wx,
                    yybOpenId: oid,
                    nickname: (yyb && yyb.nickname) || '',
                }),
            });
            toast('绑定成功', 'success');
            if (wxSel) wxSel.value = '';
            if (yybSel) yybSel.value = '';
            await afterBindChange();
        } catch (e) {
            toast(e.message || '绑定失败', 'error');
        }
    }

    async function unbindPair(wx, oid) {
        if (!global.confirm('确认解除该双绑？解绑后青龙中的 wxid 将走微信协议（若仍在线）。')) return;
        try {
            await request('/api/portal/protocol/unbind', {
                method: 'POST',
                body: JSON.stringify({ wxWxid: wx, yybOpenId: oid }),
            });
            toast('已解绑', 'success');
            await afterBindChange();
        } catch (e) {
            toast(e.message || '解绑失败', 'error');
        }
    }

    async function afterBindChange() {
        await reloadBindings();
        if (typeof global.loadWxDevices === 'function') {
            await global.loadWxDevices();
        } else if (typeof global.renderWxDevices === 'function') {
            global.renderWxDevices();
        }
        if (global.YybPortal && typeof global.YybPortal.loadPanel === 'function') {
            await global.YybPortal.loadPanel({ autoCheck: false, silent: true });
        }
    }

    function setWxDevices(devices) {
        state.wxDevices = devices || [];
        updateVisibility();
        syncBindDropdowns();
        renderBindPairList();
    }

    function setYybAccounts(accounts) {
        state.yybAccounts = accounts || [];
        updateVisibility();
        syncBindDropdowns();
        renderBindPairList();
    }

    function init() {
        document.addEventListener('click', function (e) {
            var bindBtn = e.target.closest('[data-proto-bind-submit]');
            if (bindBtn) {
                e.preventDefault();
                bindFromCard(bindBtn.closest('.proto-bind-card'));
                return;
            }
            var unbindBtn = e.target.closest('.proto-bind-unbind-btn, .proto-peer-unbind-btn');
            if (unbindBtn) {
                e.preventDefault();
                e.stopPropagation();
                unbindPair(unbindBtn.getAttribute('data-wx') || '', unbindBtn.getAttribute('data-oid') || '');
            }
        });
        void reloadBindings();
    }

    global.PortalProtocolBind = {
        init: init,
        refresh: reloadBindings,
        setWxDevices: setWxDevices,
        setYybAccounts: setYybAccounts,
        renderWxPeerStrip: renderWxPeerStrip,
        renderYybPeerStrip: renderYybPeerStrip,
        hasWxBinding: hasWxBinding,
        hasYybBinding: hasYybBinding,
        shouldShowBindUI: shouldShowBindUI,
        bindingByWx: bindingByWx,
        bindingByOpenID: bindingByOpenID,
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})(window);
