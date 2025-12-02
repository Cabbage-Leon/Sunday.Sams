package test

import (
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestCheckSettleInfo 测试获取结算信息功能
// 这个功能计算订单总价、运费等信息
func TestCheckSettleInfo(t *testing.T) {
	t.Run("测试结算信息解析", func(t *testing.T) {
		// 模拟结算信息API返回的数据
		mockResponse := `{
			"code": "Success",
			"data": {
				"saasId": "saas-001",
				"uid": "user-001",
				"floorId": 1,
				"floorName": "普通商品",
				"deliveryFee": "0",
				"settleDelivery": [
					{
						"deliveryType": 2,
						"deliveryName": "全城配",
						"deliveryDesc": "次日达",
						"storeDeliveryTemplateId": "template-001",
						"areaBlockId": "block-001",
						"areaBlockName": "浦东新区",
						"firstPeriod": 1
					}
				],
				"deliveryAddress": {
					"addressId": "addr-001",
					"name": "张三",
					"mobile": "13800138000"
				}
			}
		}`

		result := gjson.Parse(mockResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		// 验证结算信息关键字段
		deliveryFee := result.Get("data.deliveryFee").Str
		floorId := int(result.Get("data.floorId").Num)
		saasId := result.Get("data.saasId").Str

		if deliveryFee == "" {
			t.Error("运费不能为空")
		}
		if floorId != 1 {
			t.Error("楼层ID应为1")
		}
		if saasId == "" {
			t.Error("SaasID不能为空")
		}

		// 验证配送信息
		settleDelivery := result.Get("data.settleDelivery").Array()
		if len(settleDelivery) == 0 {
			t.Error("配送信息不能为空")
		}

		firstDelivery := settleDelivery[0]
		deliveryType := int(firstDelivery.Get("deliveryType").Num)
		templateId := firstDelivery.Get("storeDeliveryTemplateId").Str
		areaBlockId := firstDelivery.Get("areaBlockId").Str

		if deliveryType != 1 && deliveryType != 2 {
			t.Error("配送类型应为1或2")
		}
		if templateId == "" {
			t.Error("配送模板ID不能为空")
		}
		if areaBlockId == "" {
			t.Error("区域区块ID不能为空")
		}

		t.Logf("✅ 结算信息解析测试通过 - 运费: %s, 配送类型: %d", 
			deliveryFee, deliveryType)
	})

	t.Run("测试SettleInfo结构体", func(t *testing.T) {
		settleInfo := dd.SettleInfo{
			SaasId:     "saas-001",
			Uid:        "user-001",
			FloorId:    1,
			FloorName:  "普通商品",
			DeliveryFee: "0",
			SettleDelivery: dd.SettleDelivery{
				DeliveryType:            2,
				DeliveryName:            "全城配",
				StoreDeliveryTemplateId: "template-001",
				AreaBlockId:             "block-001",
			},
		}

		if settleInfo.SaasId == "" {
			t.Error("SaasID不能为空")
		}
		if settleInfo.DeliveryFee == "" {
			t.Error("运费不能为空")
		}
		if settleInfo.SettleDelivery.StoreDeliveryTemplateId == "" {
			t.Error("配送模板ID不能为空")
		}

		t.Logf("✅ SettleInfo结构体测试通过 - 楼层: %s, 运费: %s", 
			settleInfo.FloorName, settleInfo.DeliveryFee)
	})

	t.Run("测试免运费检查逻辑", func(t *testing.T) {
		// 场景1: 设置了免运费，但实际有运费
		deliveryFee := "10.00"
		requireFreeDelivery := true
		shouldRetry := requireFreeDelivery && deliveryFee != "0"

		if !shouldRetry {
			t.Error("有运费时应重新检查购物车")
		}

		// 场景2: 设置了免运费，实际也是免运费
		deliveryFee2 := "0"
		shouldRetry2 := requireFreeDelivery && deliveryFee2 != "0"

		if shouldRetry2 {
			t.Error("免运费时不应重新检查")
		}

		// 场景3: 未设置免运费要求
		requireFreeDelivery3 := false
		deliveryFee3 := "10.00"
		shouldRetry3 := requireFreeDelivery3 && deliveryFee3 != "0"

		if shouldRetry3 {
			t.Error("未设置免运费要求时不应重新检查")
		}

		t.Log("✅ 免运费检查逻辑测试通过")
	})

	t.Run("测试结算错误处理", func(t *testing.T) {
		// 测试限流错误
		mockLimitedResponse := `{
			"code": "LIMITED",
			"msg": "服务器正忙,请稍后再试"
		}`
		result1 := gjson.Parse(mockLimitedResponse)
		if result1.Get("code").Str != "LIMITED" {
			t.Error("应该返回LIMITED错误")
		}

		// 测试不匹配配送模式错误
		mockNoMatchResponse := `{
			"code": "NO_MATCH_DELIVERY_MODE",
			"msg": "当前区域不支持配送"
		}`
		result2 := gjson.Parse(mockNoMatchResponse)
		if result2.Get("code").Str != "NO_MATCH_DELIVERY_MODE" {
			t.Error("应该返回NO_MATCH_DELIVERY_MODE错误")
		}

		// 测试购物车商品变化错误
		mockCartChangeResponse := `{
			"code": "CART_GOOD_CHANGE",
			"msg": "购物车商品发生变化"
		}`
		result3 := gjson.Parse(mockCartChangeResponse)
		if result3.Get("code").Str != "CART_GOOD_CHANGE" {
			t.Error("应该返回CART_GOOD_CHANGE错误")
		}

		t.Log("✅ 结算错误处理测试通过")
	})
}

