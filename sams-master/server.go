package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许跨域
		},
	}
	
	// 全局状态
	globalSession *dd.DingdongSession
	sessionMutex  sync.RWMutex
	isRunning     bool
	runMutex      sync.Mutex
	logChan       chan LogMessage
	statusChan    chan StatusUpdate
)

type LogMessage struct {
	Time    string `json:"time"`
	Level   string `json:"level"` // info, success, error, warning
	Message string `json:"message"`
}

type StatusUpdate struct {
	Step        string                 `json:"step"`
	Status      string                 `json:"status"` // running, success, error, stopped
	Address     *dd.Address            `json:"address,omitempty"`
	Stores      []dd.Store             `json:"stores,omitempty"`
	GoodsList   []dd.Goods             `json:"goodsList,omitempty"`
	DeliveryFee string                 `json:"deliveryFee,omitempty"`
	TimeSlots   []dd.SettleDeliveryInfo `json:"timeSlots,omitempty"`
	Order       *dd.Order              `json:"order,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type ConfigRequest struct {
	AuthToken    string   `json:"authToken"`
	BarkId       string   `json:"barkId"`
	FloorId      int      `json:"floorId"`
	DeliveryType int      `json:"deliveryType"`
	Longitude    string   `json:"longitude"`
	Latitude     string   `json:"latitude"`
	DeviceId     string   `json:"deviceId"`
	TrackInfo    string   `json:"trackInfo"`
	PromotionId  string   `json:"promotionId"`
	AddressId    string   `json:"addressId"`
	PayMethod    int      `json:"payMethod"`
	DeliveryFee  bool     `json:"deliveryFee"`
	StoreConf    string   `json:"storeConf"`
	IsSelected   bool     `json:"isSelected"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// WebSocket连接管理
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan interface{})
var clientsMutex sync.Mutex

func init() {
	logChan = make(chan LogMessage, 100)
	statusChan = make(chan StatusUpdate, 10)
}

func logMessage(level, message string) {
	msg := LogMessage{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Message: message,
	}
	select {
	case logChan <- msg:
	default:
	}
	broadcast <- msg
}

func updateStatus(status StatusUpdate) {
	select {
	case statusChan <- status:
	default:
	}
	broadcast <- status
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	// 发送当前状态
	sessionMutex.RLock()
	if globalSession != nil {
		status := getCurrentStatus()
		conn.WriteJSON(status)
	}
	sessionMutex.RUnlock()

	// 监听广播消息
	for {
		var msg interface{}
		select {
		case msg = <-broadcast:
		case <-time.After(30 * time.Second):
			// 发送心跳
			conn.WriteJSON(map[string]string{"type": "ping"})
			continue
		}

		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("WebSocket写入错误: %v", err)
			clientsMutex.Lock()
			delete(clients, conn)
			clientsMutex.Unlock()
			break
		}
	}
}

func getCurrentStatus() StatusUpdate {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	status := StatusUpdate{
		Step:   "idle",
		Status: "stopped",
	}

	if globalSession != nil {
		status.Address = &globalSession.Address
		
		stores := make([]dd.Store, 0, len(globalSession.StoreList))
		for _, store := range globalSession.StoreList {
			stores = append(stores, store)
		}
		status.Stores = stores
		
		status.GoodsList = globalSession.GoodsList
		
		timeSlots := make([]dd.SettleDeliveryInfo, 0, len(globalSession.SettleDeliveryInfo))
		for _, slot := range globalSession.SettleDeliveryInfo {
			timeSlots = append(timeSlots, slot)
		}
		status.TimeSlots = timeSlots
	}

	if isRunning {
		status.Status = "running"
	}

	return status
}

// API处理函数
func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, APIResponse{Success: false, Message: "请求参数错误: " + err.Error()}, http.StatusBadRequest)
		return
	}

	if req.AuthToken == "" {
		respondJSON(w, APIResponse{Success: false, Message: "authToken不能为空"}, http.StatusBadRequest)
		return
	}

	splitFn := func(c rune) bool {
		return c == ','
	}

	conf := dd.Config{
		AuthToken:    req.AuthToken,
		BarkId:       req.BarkId,
		FloorId:      req.FloorId,
		DeliveryType: req.DeliveryType,
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		Deviceid:     req.DeviceId,
		Trackinfo:    req.TrackInfo,
		PromotionId:  strings.FieldsFunc(req.PromotionId, splitFn),
		AddressId:    req.AddressId,
		PayMethod:    req.PayMethod,
		DeliveryFee:  req.DeliveryFee,
		StoreConf:    req.StoreConf,
		IsSelected:   req.IsSelected,
	}

	session := &dd.DingdongSession{
		SettleDeliveryInfo: map[int]dd.SettleDeliveryInfo{},
		StoreList:          map[string]dd.Store{},
	}

	err := session.InitSession(conf)
	if err != nil {
		respondJSON(w, APIResponse{Success: false, Message: "初始化失败: " + err.Error()}, http.StatusBadRequest)
		return
	}

	// 获取地址列表（已在InitSession中获取，这里不需要再次获取）
	// err, addrList := session.GetAddress()
	// if err != nil {
	// 	respondJSON(w, APIResponse{Success: false, Message: "获取地址失败: " + err.Error()}, http.StatusBadRequest)
	// 	return
	// }
	
	// 重新获取地址列表用于返回
	err, addrList := session.GetAddress()
	if err != nil {
		respondJSON(w, APIResponse{Success: false, Message: "获取地址失败: " + err.Error()}, http.StatusBadRequest)
		return
	}

	sessionMutex.Lock()
	globalSession = session
	sessionMutex.Unlock()

	logMessage("success", "配置保存成功")
	updateStatus(StatusUpdate{
		Step:   "configured",
		Status: "stopped",
		Address: &session.Address,
	})

	respondJSON(w, APIResponse{
		Success: true,
		Message: "配置成功",
		Data: map[string]interface{}{
			"addressList": addrList,
			"selectedAddress": session.Address,
		},
	}, http.StatusOK)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runMutex.Lock()
	if isRunning {
		runMutex.Unlock()
		respondJSON(w, APIResponse{Success: false, Message: "程序已在运行中"}, http.StatusBadRequest)
		return
	}

	sessionMutex.RLock()
	if globalSession == nil {
		sessionMutex.RUnlock()
		runMutex.Unlock()
		respondJSON(w, APIResponse{Success: false, Message: "请先配置参数"}, http.StatusBadRequest)
		return
	}
	sessionMutex.RUnlock()

	isRunning = true
	runMutex.Unlock()

	logMessage("info", "开始执行抢购流程...")
	updateStatus(StatusUpdate{
		Step:   "starting",
		Status: "running",
	})

	// 在goroutine中运行主流程
	go runMainLoop()

	respondJSON(w, APIResponse{Success: true, Message: "已开始执行"}, http.StatusOK)
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runMutex.Lock()
	isRunning = false
	runMutex.Unlock()

	logMessage("warning", "用户手动停止")
	updateStatus(StatusUpdate{
		Step:   "stopped",
		Status: "stopped",
	})

	respondJSON(w, APIResponse{Success: true, Message: "已停止"}, http.StatusOK)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := getCurrentStatus()
	respondJSON(w, APIResponse{Success: true, Data: status}, http.StatusOK)
}

func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// 主循环（从main.go移植过来，但添加了状态更新）
func runMainLoop() {
	defer func() {
		runMutex.Lock()
		isRunning = false
		runMutex.Unlock()
	}()

	sessionMutex.RLock()
	session := globalSession
	sessionMutex.RUnlock()

	if session == nil {
		logMessage("error", "会话未初始化")
		return
	}

	for {
		runMutex.Lock()
		if !isRunning {
			runMutex.Unlock()
			return
		}
		runMutex.Unlock()

	SaveDeliveryAddress:
		logMessage("info", "切换购物车收货地址...")
		updateStatus(StatusUpdate{Step: "saving_address", Status: "running"})
		
		err := session.SaveDeliveryAddress()
		if err != nil {
			logMessage("error", "保存地址失败: "+err.Error())
			time.Sleep(1 * time.Second)
			goto SaveDeliveryAddress
		} else {
			logMessage("success", fmt.Sprintf("地址保存成功: %s %s %s", 
				session.Address.DistrictName, session.Address.ReceiverAddress, session.Address.DetailAddress))
			updateStatus(StatusUpdate{
				Step:    "address_saved",
				Status:  "running",
				Address: &session.Address,
			})
		}

		if session.Conf.StoreConf != "" {
			if _, err := os.Stat(session.Conf.StoreConf); err == nil {
				if file, err := os.Open(session.Conf.StoreConf); err == nil {
					logMessage("info", "预加载商店配置...")
					var bytes []byte
					buf := make([]byte, 1024)
					for {
						n, err := file.Read(buf)
						if err != nil && err != io.EOF {
							logMessage("error", "读取文件失败: "+err.Error())
							file.Close()
							return
						}
						if n == 0 {
							break
						}
						bytes = append(bytes, buf[:n]...)
					}

					for _, store := range session.GetStoreList(gjson.ParseBytes(bytes)) {
						if _, ok := session.StoreList[store.StoreId]; !ok {
							session.StoreList[store.StoreId] = store
							logMessage("info", fmt.Sprintf("加载商店: %s", store.StoreName))
						}
					}
					file.Close()
				}
			}
		}

	StoreLoop:
		logMessage("info", "获取地址附近可用商店...")
		updateStatus(StatusUpdate{Step: "checking_stores", Status: "running"})
		
		stores, err := session.CheckStore()
		if err != nil {
			logMessage("error", "获取商店失败: "+err.Error())
			time.Sleep(1 * time.Second)
			goto StoreLoop
		}

		storeList := make([]dd.Store, 0, len(stores))
		for _, store := range stores {
			if oStore, ok := session.StoreList[store.StoreId]; !ok || oStore.StoreDeliveryTemplateId != store.StoreDeliveryTemplateId || oStore.AreaBlockId != store.AreaBlockId {
				session.StoreList[store.StoreId] = store
				storeList = append(storeList, store)
				logMessage("info", fmt.Sprintf("发现商店: %s", store.StoreName))
			}
		}

		updateStatus(StatusUpdate{
			Step:   "stores_loaded",
			Status: "running",
			Stores: storeList,
		})

	CartLoop:
		logMessage("info", fmt.Sprintf("获取购物车中有效商品【%s】...", time.Now().Format("15:04:05")))
		updateStatus(StatusUpdate{Step: "checking_cart", Status: "running"})
		
		err = session.CheckCart()
		for _, v := range session.Cart.FloorInfoList {
			if v.FloorId == session.Conf.FloorId && v.DeliveryType == session.Conf.DeliveryType {
				session.GoodsList = make([]dd.Goods, 0)
				for _, goods := range v.NormalGoodsList {
					if goods.StockQuantity > 0 && goods.StockStatus && goods.IsPutOnSale && goods.IsAvailable {
						if goods.StockQuantity <= goods.Quantity {
							goods.Quantity = goods.StockQuantity
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.LimitNum {
							goods.Quantity = goods.LimitNum
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.ResiduePurchaseNum {
							goods.Quantity = goods.ResiduePurchaseNum
						}
						if goods.Quantity > 0 {
							session.GoodsList = append(session.GoodsList, goods.ToGoods())
						}
					}
				}

				for _, goods := range v.ShortageStockGoodsList {
					if goods.StockQuantity > 0 && goods.StockStatus && goods.IsPutOnSale && goods.IsAvailable {
						if goods.StockQuantity <= goods.Quantity {
							goods.Quantity = goods.StockQuantity
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.LimitNum {
							goods.Quantity = goods.LimitNum
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.ResiduePurchaseNum {
							goods.Quantity = goods.ResiduePurchaseNum
						}
						if goods.Quantity > 0 {
							session.GoodsList = append(session.GoodsList, goods.ToGoods())
						}
					}
				}

				for _, goods := range v.AllOutOfStockGoodsList {
					if goods.StockQuantity > 0 && goods.StockStatus && goods.IsPutOnSale && goods.IsAvailable {
						if goods.StockQuantity <= goods.Quantity {
							goods.Quantity = goods.StockQuantity
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.LimitNum {
							goods.Quantity = goods.LimitNum
						}
						if goods.LimitNum > 0 && goods.Quantity > goods.ResiduePurchaseNum {
							goods.Quantity = goods.ResiduePurchaseNum
						}
						if goods.Quantity > 0 {
							session.GoodsList = append(session.GoodsList, goods.ToGoods())
						}
					}
				}

				session.FloorInfo = v
			}
		}

		var selGoods = make([]dd.Goods, 0)
		for _, goods := range session.GoodsList {
			logMessage("info", fmt.Sprintf("商品: %s 数量: %d 价格: %d", goods.GoodsName, goods.Quantity, goods.Price))
			if goods.IsSelected && session.Conf.IsSelected {
				selGoods = append(selGoods, goods)
			}
		}

		if session.Conf.IsSelected {
			session.GoodsList = selGoods
		}

		if len(session.GoodsList) == 0 {
			logMessage("warning", "当前购物车中无有效商品")
			if errors.Is(err, dd.LimitedErr1) {
				time.Sleep(1 * time.Second)
			}
			goto StoreLoop
		}

		updateStatus(StatusUpdate{
			Step:      "cart_loaded",
			Status:    "running",
			GoodsList: session.GoodsList,
		})

	GoodsLoop:
		logMessage("info", fmt.Sprintf("开始校验当前商品【%s】...", time.Now().Format("15:04:05")))
		updateStatus(StatusUpdate{Step: "checking_goods", Status: "running"})
		
		if _, err := session.CheckGoods(); err != nil {
			logMessage("error", "商品校验失败: "+err.Error())
			time.Sleep(1 * time.Second)
			switch err {
			case dd.OOSErr:
				goto CartLoop
			default:
				goto CartLoop
			}
		}

		if settleInfo, err := session.CheckSettleInfo(); err == nil {
			logMessage("info", fmt.Sprintf("运费: %s", settleInfo.DeliveryFee))
			updateStatus(StatusUpdate{
				Step:        "settle_checked",
				Status:      "running",
				DeliveryFee: settleInfo.DeliveryFee,
			})

			if store, ok := session.StoreList[session.FloorInfo.StoreId]; ok && store.StoreDeliveryTemplateId != settleInfo.SettleDelivery.StoreDeliveryTemplateId {
				store.StoreDeliveryTemplateId = settleInfo.SettleDelivery.StoreDeliveryTemplateId
				store.AreaBlockId = settleInfo.SettleDelivery.AreaBlockId
				session.StoreList[session.FloorInfo.StoreId] = store
			}

			if session.Conf.DeliveryFee && settleInfo.DeliveryFee != "0" {
				logMessage("warning", "需要运费，重新检查购物车")
				goto CartLoop
			}
		} else {
			logMessage("error", "校验商品失败: "+err.Error())
			time.Sleep(1 * time.Second)
			switch err {
			case dd.CartGoodChangeErr:
				goto CartLoop
			case dd.LimitedErr:
				goto GoodsLoop
			case dd.NoMatchDeliverMode:
				goto SaveDeliveryAddress
			default:
				goto GoodsLoop
			}
		}

	CapacityLoop:
		logMessage("info", fmt.Sprintf("获取当前可用配送时间【%s】...", time.Now().Format("15:04:05")))
		updateStatus(StatusUpdate{Step: "checking_capacity", Status: "running"})
		
		capacity, err := session.GetCapacity(session.StoreList[session.FloorInfo.StoreId].StoreDeliveryTemplateId)
		if err != nil {
			logMessage("error", "获取配送时间失败: "+err.Error())
			switch err {
			case dd.CapacityErr:
				goto StoreLoop
			default:
				time.Sleep(1 * time.Second)
				goto CapacityLoop
			}
		}

		session.SettleDeliveryInfo = map[int]dd.SettleDeliveryInfo{}
		for _, caps := range capacity.CapCityResponseList {
			for _, v := range caps.List {
				if v.TimeISFull == false && v.Disabled == false {
					session.SettleDeliveryInfo[len(session.SettleDeliveryInfo)] = dd.SettleDeliveryInfo{
						ArrivalTimeStr:       fmt.Sprintf("%s %s - %s", caps.StrDate, v.StartTime, v.EndTime),
						ExpectArrivalTime:    v.StartRealTime,
						ExpectArrivalEndTime: v.EndRealTime,
					}
				}
			}
		}

		timeSlots := make([]dd.SettleDeliveryInfo, 0, len(session.SettleDeliveryInfo))
		for _, v := range session.SettleDeliveryInfo {
			timeSlots = append(timeSlots, v)
			logMessage("success", "发现可用配送时段: "+v.ArrivalTimeStr)
		}

		if len(session.SettleDeliveryInfo) == 0 {
			logMessage("warning", "当前无可用配送时间段")
			time.Sleep(1 * time.Second)
			goto CapacityLoop
		}

		updateStatus(StatusUpdate{
			Step:      "capacity_loaded",
			Status:    "running",
			TimeSlots: timeSlots,
		})

	OrderLoop:
		for len(session.SettleDeliveryInfo) > 0 {
			runMutex.Lock()
			if !isRunning {
				runMutex.Unlock()
				return
			}
			runMutex.Unlock()

			for k, v := range session.SettleDeliveryInfo {
				logMessage("info", fmt.Sprintf("提交订单中【%s】配送时段: %s", time.Now().Format("15:04:05"), v.ArrivalTimeStr))
				updateStatus(StatusUpdate{Step: "submitting_order", Status: "running"})
				
				if order, err := session.CommitPay(v); err == nil {
					logMessage("success", fmt.Sprintf("抢购成功！订单号: %s，请前往app付款！", order.OrderNo))
					updateStatus(StatusUpdate{
						Step:   "order_success",
						Status: "success",
						Order:  order,
					})

					if session.Conf.BarkId != "" {
						for {
							err = session.PushSuccess(fmt.Sprintf("Smas抢单成功，订单号：%s", order.OrderNo))
							if err == nil {
								break
							}
							time.Sleep(1 * time.Second)
						}
					}

					runMutex.Lock()
					isRunning = false
					runMutex.Unlock()
					return
				} else {
					logMessage("error", "下单失败: "+err.Error())
					switch err {
					case dd.LimitedErr1:
						logMessage("info", "立即重试...")
						goto OrderLoop
					case dd.CloudGoodsOverWightErr:
						maxKey := len(session.GoodsList) - 1
						for key, v := range session.GoodsList {
							if v.Quantity > 1 && v.Weight > session.GoodsList[maxKey].Weight {
								maxKey = key
							}
						}
						if maxKey >= 0 {
							if session.GoodsList[maxKey].Quantity > 1 {
								session.GoodsList[maxKey].Quantity -= 1
							} else {
								session.GoodsList = append(session.GoodsList[:maxKey], session.GoodsList[maxKey+1:]...)
							}
						}
						goto OrderLoop
					case dd.OOSErr, dd.PreGoodNotStartSellErr, dd.CartGoodChangeErr, dd.GoodsExceedLimitErr:
						goto CartLoop
					case dd.StoreHasClosedError, dd.GetDeliveryInfoErr:
						goto StoreLoop
					case dd.CloseOrderTimeExceptionErr, dd.DecreaseCapacityCountError, dd.NotDeliverCapCityErr:
						delete(session.SettleDeliveryInfo, k)
					default:
						goto CapacityLoop
					}
				}
			}
		}
		goto CapacityLoop
	}
}

func main() {
	// 检查是否以服务器模式运行
	if len(os.Args) > 1 && os.Args[1] == "server" {
		startServer()
		return
	}

	// 默认运行原来的命令行模式
	// 这里可以保留原来的main逻辑或提示用户
	fmt.Println("使用 'go run . server' 启动Web服务器")
	fmt.Println("或使用原来的命令行参数运行")
}

func startServer() {
	port := "8080"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	// 静态文件服务
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// API路由
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/api/start", handleStart)
	http.HandleFunc("/api/stop", handleStop)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/ws", handleWebSocket)

	log.Printf("🚀 服务器启动在 http://localhost:%s", port)
	log.Printf("📱 打开浏览器访问 http://localhost:%s 使用可视化界面", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

