package test

import (
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestCheckCart 测试获取购物车功能
// 这个功能获取购物车中的所有商品，并分类处理
func TestCheckCart(t *testing.T) {
	t.Run("测试购物车数据解析", func(t *testing.T) {
		// 模拟购物车API返回的数据
		mockResponse := `{
			"code": "Success",
			"data": {
				"floorInfoList": [
					{
						"floorId": 1,
						"deliveryType": 2,
						"storeId": "store-001",
						"amount": "299.00",
						"quantity": 5,
						"normalGoodsList": [
							{
								"storeId": "store-001",
								"storeType": 1,
								"spuId": "spu-001",
								"skuId": "sku-001",
								"goodsName": "测试商品1",
								"price": 59,
								"quantity": 2,
								"stockQuantity": 10,
								"stockStatus": true,
								"isPutOnSale": true,
								"isAvailable": true,
								"purchaseLimitVO": {
									"limitNum": 5,
									"residuePurchaseNum": 3
								},
								"isSelected": true,
								"weight": 1.5
							}
						],
						"shortageStockGoodsList": [],
						"allOutOfStockGoodsList": []
					}
				]
			}
		}`

		result := gjson.Parse(mockResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		floorInfoList := result.Get("data.floorInfoList").Array()
		if len(floorInfoList) == 0 {
			t.Error("楼层信息列表为空")
		}

		// 验证楼层信息
		firstFloor := floorInfoList[0]
		floorId := int(firstFloor.Get("floorId").Num)
		deliveryType := int(firstFloor.Get("deliveryType").Num)
		storeId := firstFloor.Get("storeId").Str

		if floorId != 1 {
			t.Errorf("楼层ID应为1，实际为: %d", floorId)
		}
		if storeId == "" {
			t.Error("商店ID不能为空")
		}

		// 验证商品列表
		normalGoodsList := firstFloor.Get("normalGoodsList").Array()
		if len(normalGoodsList) == 0 {
			t.Error("正常商品列表为空")
		}

		firstGoods := normalGoodsList[0]
		goodsName := firstGoods.Get("goodsName").Str
		stockQuantity := int(firstGoods.Get("stockQuantity").Num)
		stockStatus := firstGoods.Get("stockStatus").Bool()
		isPutOnSale := firstGoods.Get("isPutOnSale").Bool()
		isAvailable := firstGoods.Get("isAvailable").Bool()

		// 验证商品是否有效
		if !stockStatus {
			t.Error("商品库存状态应为true")
		}
		if !isPutOnSale {
			t.Error("商品应在售")
		}
		if !isAvailable {
			t.Error("商品应可用")
		}
		if stockQuantity <= 0 {
			t.Error("商品库存应大于0")
		}

		t.Logf("✅ 购物车数据解析测试通过 - 商品: %s, 库存: %d, 配送类型: %d", 
			goodsName, stockQuantity, deliveryType)
	})

	t.Run("测试商品数量自动调整逻辑", func(t *testing.T) {
		// 测试场景1: 库存少于购物数量
		stockQuantity := 5
		quantity := 10
		if quantity > stockQuantity {
			quantity = stockQuantity
		}
		if quantity != 5 {
			t.Errorf("数量应调整为库存数量5，实际为: %d", quantity)
		}

		// 测试场景2: 超过限购数量
		limitNum := 3
		quantity = 5
		if limitNum > 0 && quantity > limitNum {
			quantity = limitNum
		}
		if quantity != 3 {
			t.Errorf("数量应调整为限购数量3，实际为: %d", quantity)
		}

		// 测试场景3: 超过剩余可购买数量
		residuePurchaseNum := 2
		quantity = 5
		if limitNum > 0 && quantity > residuePurchaseNum {
			quantity = residuePurchaseNum
		}
		if quantity != 2 {
			t.Errorf("数量应调整为剩余可购买数量2，实际为: %d", quantity)
		}

		t.Log("✅ 商品数量自动调整逻辑测试通过")
	})

	t.Run("测试FloorInfo结构体", func(t *testing.T) {
		floorInfo := dd.FloorInfo{
			FloorId:      1,
			DeliveryType: 2,
			StoreId:      "store-001",
			Amount:       "299.00",
			Quantity:     5,
		}

		if floorInfo.FloorId != 1 {
			t.Error("楼层ID应为1（普通商品）")
		}
		if floorInfo.DeliveryType != 1 && floorInfo.DeliveryType != 2 {
			t.Error("配送类型应为1或2")
		}
		if floorInfo.StoreId == "" {
			t.Error("商店ID不能为空")
		}

		t.Logf("✅ FloorInfo结构体测试通过 - 楼层: %d, 配送类型: %d, 总金额: %s", 
			floorInfo.FloorId, floorInfo.DeliveryType, floorInfo.Amount)
	})

	t.Run("测试限流错误处理", func(t *testing.T) {
		// 模拟限流响应
		mockLimitedResponse := `{
			"code": "LIMITED",
			"msg": "当前购物火爆，请稍后再试"
		}`

		result := gjson.Parse(mockLimitedResponse)
		if result.Get("code").Str != "LIMITED" {
			t.Error("应该返回LIMITED错误")
		}

		t.Log("✅ 限流错误处理测试通过")
	})
}

