(function (global) {
    'use strict';

    async function loadOptions(activityId, remarks) {
        var url = '/api/portal/protocol/account-options?activityId=' + encodeURIComponent(activityId || '');
        if (remarks) {
            url += '&remarks=' + encodeURIComponent(remarks);
        }
        var resp = await request(url);
        return resp.data || [];
    }

    function findOptionByFillRef(options, fillRef) {
        fillRef = String(fillRef || '').trim();
        if (!fillRef) return null;
        return (options || []).find(function (opt) {
            return String(opt.fillRef || '').trim() === fillRef
                || String(opt.openid || '').trim() === fillRef
                || String(opt.wxid || '').trim() === fillRef;
        }) || null;
    }

    function firstTemplateFieldKey(template) {
        var fields = (global.getCkTemplateFields ? global.getCkTemplateFields(template) : []);
        return fields.length ? fields[0] : '';
    }

    function applyFillRef(fillRef, template, container, selector) {
        var key = firstTemplateFieldKey(template);
        if (!key) return;
        var root = container;
        if (typeof container === 'string') {
            root = document.querySelector(container);
        }
        if (!root) return;
        var sel = selector || ('[data-modal-field-key="' + key + '"],[data-edit-ck-key="' + key + '"]');
        var input = root.querySelector(sel);
        if (input) {
            input.value = fillRef || '';
            input.dispatchEvent(new Event('input', { bubbles: true }));
        }
    }

    function renderSelect(selectEl, options, selectedFillRef) {
        if (!selectEl) return;
        var current = String(selectedFillRef || '').trim();
        selectEl.innerHTML = '';
        var ph = document.createElement('option');
        ph.value = '';
        ph.textContent = '请选择协议账号';
        selectEl.appendChild(ph);
        (options || []).forEach(function (opt) {
            var item = document.createElement('option');
            item.value = String(opt.fillRef || '');
            item.textContent = String(opt.label || opt.nickname || opt.fillRef || '协议账号');
            item.disabled = opt.selectable === false;
            if (opt.usedInActivity && opt.selectable === false) {
                item.style.color = '#9ca3af';
            }
            if (current && (item.value === current || findOptionByFillRef([opt], current))) {
                item.selected = true;
            }
            selectEl.appendChild(item);
        });
        if (current) {
            var matched = Array.from(selectEl.options).find(function (o) { return o.value === current && !o.disabled; });
            if (matched) selectEl.value = current;
        }
    }

    function mountPicker(config) {
        var container = config.container;
        if (!container) return Promise.resolve(null);
        var activityId = config.activityId;
        var remarks = config.remarks || '';
        var template = config.template || '';
        var selectedFillRef = config.selectedFillRef || '';
        var fieldKey = firstTemplateFieldKey(template);
        return loadOptions(activityId, remarks).then(function (options) {
            var wrap = document.createElement('div');
            wrap.className = 'field full proto-activity-picker';
            wrap.innerHTML = ''
                + '<label>协议账号<span class="field-required" title="必选">*</span></label>'
                + '<select data-proto-activity-select></select>'
                + '<div class="proto-activity-hint" style="display:none;margin-top:6px;font-size:11px;color:#9ca3af;"></div>';
            var selectEl = wrap.querySelector('[data-proto-activity-select]');
            var hintEl = wrap.querySelector('.proto-activity-hint');
            renderSelect(selectEl, options, selectedFillRef);
            if (!options.length) {
                hintEl.style.display = 'block';
                hintEl.textContent = '暂无在线协议账号，请先到「协议接入」扫码登录。';
            } else if (!options.some(function (o) { return o.selectable !== false; })) {
                hintEl.style.display = 'block';
                hintEl.textContent = '本活动在线协议账号已全部上车。';
            }
            selectEl.addEventListener('change', function () {
                applyFillRef(selectEl.value, template, container);
                if (typeof config.onChange === 'function') config.onChange(selectEl.value);
            });
            if (selectEl.value) {
                applyFillRef(selectEl.value, template, container);
            }
            container.insertBefore(wrap, container.firstChild);
            if (fieldKey) {
                var input = container.querySelector('[data-modal-field-key="' + fieldKey + '"],[data-edit-ck-key="' + fieldKey + '"]');
                if (input) {
                    var row = input.closest('.field') || input.closest('.ck-field-row');
                    if (row) row.style.display = 'none';
                }
            }
            return { wrap: wrap, selectEl: selectEl, options: options, fieldKey: fieldKey };
        });
    }

    global.PortalProtocolActivity = {
        loadOptions: loadOptions,
        renderSelect: renderSelect,
        applyFillRef: applyFillRef,
        findOptionByFillRef: findOptionByFillRef,
        mountPicker: mountPicker,
        firstTemplateFieldKey: firstTemplateFieldKey,
    };
})(window);
