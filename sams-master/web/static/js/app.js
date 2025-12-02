// WebSocket连接
let ws = null;
let reconnectTimer = null;

// 状态管理
const state = {
    isRunning: false,
    currentStep: 'idle',
    address: null,
    stores: [],
    goodsList: [],
    timeSlots: [],
    order: null
};

// 步骤映射
const stepMap = {
    'idle': { title: '等待开始', desc: '配置参数后点击开始', icon: '⏸️' },
    'configured': { title: '配置完成', desc: '参数已保存', icon: '✅' },
    'starting': { title: '正在启动', desc: '初始化中...', icon: '🚀' },
    'saving_address': { title: '保存地址', desc: '正在保存配送地址...', icon: '📍' },
    'address_saved': { title: '地址已保存', desc: '配送地址设置成功', icon: '✅' },
    'checking_stores': { title: '查找商店', desc: '正在查找可用门店...', icon: '🏪' },
    'stores_loaded': { title: '商店已加载', desc: '已找到可用门店', icon: '✅' },
    'checking_cart': { title: '检查购物车', desc: '正在获取购物车商品...', icon: '🛒' },
    'cart_loaded': { title: '购物车已加载', desc: '已获取购物车商品', icon: '✅' },
    'checking_goods': { title: '校验商品', desc: '正在校验商品状态...', icon: '🔍' },
    'settle_checked': { title: '结算信息', desc: '正在计算运费...', icon: '💰' },
    'checking_capacity': { title: '获取配送时间', desc: '正在查询可用时间段...', icon: '⏰' },
    'capacity_loaded': { title: '配送时间已获取', desc: '已找到可用时间段', icon: '✅' },
    'submitting_order': { title: '提交订单', desc: '正在提交订单...', icon: '📦' },
    'order_success': { title: '订单成功', desc: '抢购成功！', icon: '🎉' },
    'stopped': { title: '已停止', desc: '程序已停止', icon: '⏹️' }
};

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    initEventListeners();
    loadStatus();
});

// 初始化WebSocket
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket连接已建立');
        addLog('info', '已连接到服务器');
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWebSocketMessage(data);
    };

    ws.onerror = (error) => {
        console.error('WebSocket错误:', error);
        addLog('error', 'WebSocket连接错误');
    };

    ws.onclose = () => {
        console.log('WebSocket连接已关闭');
        addLog('warning', '连接已断开，正在重连...');
        // 5秒后重连
        reconnectTimer = setTimeout(initWebSocket, 5000);
    };
}

// 处理WebSocket消息
function handleWebSocketMessage(data) {
    if (data.type === 'ping') {
        return;
    }

    // 日志消息
    if (data.time && data.level && data.message) {
        addLog(data.level, data.message);
    }

    // 状态更新
    if (data.step !== undefined) {
        updateStatus(data);
    }
}

// 初始化事件监听
function initEventListeners() {
    // 配置表单提交
    document.getElementById('configForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        await saveConfig();
    });

    // 开始按钮
    document.getElementById('startBtn').addEventListener('click', async () => {
        await startProcess();
    });

    // 停止按钮
    document.getElementById('stopBtn').addEventListener('click', async () => {
        await stopProcess();
    });

    // 清空日志
    document.getElementById('clearLogBtn').addEventListener('click', () => {
        document.getElementById('logContainer').innerHTML = '';
    });
}

// 保存配置
async function saveConfig() {
    const formData = new FormData(document.getElementById('configForm'));
    const config = {
        authToken: formData.get('authToken'),
        addressId: formData.get('addressId') || '',
        deliveryType: parseInt(formData.get('deliveryType')) || 2,
        payMethod: parseInt(formData.get('payMethod')) || 1,
        floorId: parseInt(formData.get('floorId')) || 1,
        barkId: formData.get('barkId') || '',
        longitude: formData.get('longitude') || '',
        latitude: formData.get('latitude') || '',
        promotionId: formData.get('promotionId') || '',
        deliveryFee: formData.get('deliveryFee') === 'on',
        isSelected: formData.get('isSelected') === 'on',
        deviceId: '',
        trackInfo: '',
        storeConf: ''
    };

    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(config)
        });

        const result = await response.json();
        
        if (result.success) {
            addLog('success', '配置保存成功');
            if (result.data && result.data.selectedAddress) {
                displayAddress(result.data.selectedAddress);
            }
            if (result.data && result.data.addressList) {
                // 可以显示地址列表供选择
                console.log('地址列表:', result.data.addressList);
            }
            document.getElementById('startBtn').disabled = false;
        } else {
            addLog('error', '配置保存失败: ' + result.message);
            alert('配置失败: ' + result.message);
        }
    } catch (error) {
        addLog('error', '请求失败: ' + error.message);
        alert('请求失败: ' + error.message);
    }
}

// 开始流程
async function startProcess() {
    try {
        const response = await fetch('/api/start', {
            method: 'POST'
        });

        const result = await response.json();
        
        if (result.success) {
            state.isRunning = true;
            updateUI();
            addLog('success', '已开始执行抢购流程');
        } else {
            addLog('error', '启动失败: ' + result.message);
            alert('启动失败: ' + result.message);
        }
    } catch (error) {
        addLog('error', '请求失败: ' + error.message);
        alert('请求失败: ' + error.message);
    }
}

// 停止流程
async function stopProcess() {
    try {
        const response = await fetch('/api/stop', {
            method: 'POST'
        });

        const result = await response.json();
        
        if (result.success) {
            state.isRunning = false;
            updateUI();
            addLog('warning', '已停止执行');
        } else {
            addLog('error', '停止失败: ' + result.message);
        }
    } catch (error) {
        addLog('error', '请求失败: ' + error.message);
    }
}

// 加载状态
async function loadStatus() {
    try {
        const response = await fetch('/api/status');
        const result = await response.json();
        
        if (result.success && result.data) {
            updateStatus(result.data);
        }
    } catch (error) {
        console.error('加载状态失败:', error);
    }
}

// 更新状态
function updateStatus(data) {
    if (data.step) {
        state.currentStep = data.step;
    }
    if (data.status) {
        state.isRunning = data.status === 'running';
    }
    if (data.address) {
        state.address = data.address;
        displayAddress(data.address);
    }
    if (data.stores) {
        state.stores = data.stores;
    }
    if (data.goodsList) {
        state.goodsList = data.goodsList;
        displayGoods(data.goodsList);
    }
    if (data.timeSlots) {
        state.timeSlots = data.timeSlots;
        displayTimeSlots(data.timeSlots);
    }
    if (data.order) {
        state.order = data.order;
        displayOrder(data.order);
    }
    if (data.error) {
        addLog('error', data.error);
    }

    updateUI();
}

// 更新UI
function updateUI() {
    // 更新状态指示器
    const statusDot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');
    
    if (state.isRunning) {
        statusDot.className = 'status-dot running';
        statusText.textContent = '运行中';
        document.getElementById('startBtn').disabled = true;
        document.getElementById('stopBtn').disabled = false;
    } else if (state.currentStep === 'order_success') {
        statusDot.className = 'status-dot success';
        statusText.textContent = '抢购成功';
        document.getElementById('startBtn').disabled = true;
        document.getElementById('stopBtn').disabled = false;
    } else {
        statusDot.className = 'status-dot stopped';
        statusText.textContent = '未运行';
        document.getElementById('startBtn').disabled = !state.address;
        document.getElementById('stopBtn').disabled = true;
    }

    // 更新步骤显示
    updateSteps();
}

// 更新步骤显示
function updateSteps() {
    const container = document.getElementById('stepsContainer');
    const step = stepMap[state.currentStep] || stepMap['idle'];
    
    container.innerHTML = `
        <div class="step ${state.isRunning ? 'active' : ''} ${state.currentStep === 'order_success' ? 'success' : ''}">
            <div class="step-icon">${step.icon}</div>
            <div class="step-content">
                <div class="step-title">${step.title}</div>
                <div class="step-desc">${step.desc}</div>
            </div>
        </div>
    `;
}

// 显示地址
function displayAddress(address) {
    const panel = document.getElementById('addressPanel');
    const info = document.getElementById('addressInfo');
    
    panel.style.display = 'block';
    info.innerHTML = `
        <div class="address-info">
            <div class="address-line"><strong>收货人:</strong> ${address.name}</div>
            <div class="address-line"><strong>电话:</strong> ${address.mobile}</div>
            <div class="address-line"><strong>地址:</strong> ${address.districtName} ${address.receiverAddress} ${address.detailAddress}</div>
        </div>
    `;
}

// 显示商品列表
function displayGoods(goodsList) {
    if (!goodsList || goodsList.length === 0) {
        document.getElementById('goodsPanel').style.display = 'none';
        return;
    }

    const panel = document.getElementById('goodsPanel');
    const list = document.getElementById('goodsList');
    
    panel.style.display = 'block';
    list.innerHTML = goodsList.map(goods => `
        <div class="goods-item">
            <div class="goods-name">${goods.goodsName || '未知商品'}</div>
            <div class="goods-info">
                <span>数量: ${goods.quantity}</span>
                <span>单价: ¥${(goods.price / 100).toFixed(2)}</span>
                <span>总价: ¥${(goods.price * goods.quantity / 100).toFixed(2)}</span>
            </div>
        </div>
    `).join('');
}

// 显示配送时间
function displayTimeSlots(timeSlots) {
    if (!timeSlots || timeSlots.length === 0) {
        document.getElementById('timeSlotsPanel').style.display = 'none';
        return;
    }

    const panel = document.getElementById('timeSlotsPanel');
    const list = document.getElementById('timeSlotsList');
    
    panel.style.display = 'block';
    list.innerHTML = timeSlots.map(slot => `
        <div class="time-slot">
            <div class="time-slot-text">${slot.arrivalTimeStr}</div>
        </div>
    `).join('');
}

// 显示订单信息
function displayOrder(order) {
    if (!order) {
        document.getElementById('orderPanel').style.display = 'none';
        return;
    }

    const panel = document.getElementById('orderPanel');
    const info = document.getElementById('orderInfo');
    
    panel.style.display = 'block';
    info.innerHTML = `
        <div class="order-info">
            <div class="order-success">🎉 抢购成功！</div>
            <div class="order-detail"><strong>订单号:</strong> ${order.orderNo}</div>
            <div class="order-detail"><strong>支付金额:</strong> ¥${order.payAmount}</div>
            <div class="order-detail"><strong>支付方式:</strong> ${order.channel === 'wechat' ? '微信支付' : '支付宝'}</div>
            <div class="order-detail" style="margin-top: 15px; color: #4CAF50; font-weight: 600;">
                请前往山姆APP完成支付！
            </div>
        </div>
    `;
}

// 添加日志
function addLog(level, message) {
    const container = document.getElementById('logContainer');
    const time = new Date().toLocaleTimeString('zh-CN');
    
    const entry = document.createElement('div');
    entry.className = 'log-entry';
    entry.innerHTML = `
        <span class="log-time">${time}</span>
        <span class="log-level ${level}">${level.toUpperCase()}</span>
        <span class="log-message">${escapeHtml(message)}</span>
    `;
    
    container.appendChild(entry);
    container.scrollTop = container.scrollHeight;
}

// HTML转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

