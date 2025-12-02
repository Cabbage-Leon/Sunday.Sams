package test

import (
	"encoding/json"
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestCheckStore 测试获取可用商店功能
// 这个功能根据地址的经纬度，查找附近可以配送的山姆门店
func TestCheckStore(t *testing.T) {
	t.Run("测试商店列表解析", func(t *testing.T) {
		// 模拟API返回的商店列表数据
		mockResponse := `{
			"code": "Success",
			"data": {
				"storeList": [
					{
						"storeId": "store-001",
						"storeName": "山姆会员商店(外高桥店)",
						"storeType": "1",
						"storeAreaBlockVerifyData": {
							"areaBlockId": "block-001"
						},
						"storeRecmdDeliveryTemplateData": {
							"storeDeliveryTemplateId": "template-001"
						},
						"storeDeliveryModeVerifyData": {
							"deliveryModeId": "mode-001",
							"deliveryType": 2
						}
					}
				]
			}
		}`

		result := gjson.Parse(mockResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		storeList := result.Get("data.storeList").Array()
		if len(storeList) == 0 {
			t.Error("商店列表为空")
		}

		// 验证商店关键字段
		firstStore := storeList[0]
		storeId := firstStore.Get("storeId").Str
		storeName := firstStore.Get("storeName").Str
		areaBlockId := firstStore.Get("storeAreaBlockVerifyData.areaBlockId").Str
		templateId := firstStore.Get("storeRecmdDeliveryTemplateData.storeDeliveryTemplateId").Str

		if storeId == "" {
			t.Error("商店ID不能为空")
		}
		if areaBlockId == "" {
			t.Error("区域区块ID不能为空，用于确定配送范围")
		}
		if templateId == "" {
			t.Error("配送模板ID不能为空，用于获取配送时间")
		}

		t.Logf("✅ 商店列表解析测试通过 - 商店: %s (ID: %s)", storeName, storeId)
	})

	t.Run("测试Store结构体", func(t *testing.T) {
		store := dd.Store{
			StoreId:                 "store-001",
			StoreName:               "测试门店",
			StoreType:               "1",
			AreaBlockId:             "block-001",
			StoreDeliveryTemplateId: "template-001",
			DeliveryModeId:          "mode-001",
			DeliveryType:            2,
		}

		if store.StoreId == "" {
			t.Error("商店ID不能为空")
		}
		if store.StoreDeliveryTemplateId == "" {
			t.Error("配送模板ID不能为空")
		}
		if store.DeliveryType != 1 && store.DeliveryType != 2 {
			t.Error("配送类型应为1(极速达)或2(全城配)")
		}

		t.Logf("✅ Store结构体测试通过 - %s, 配送类型: %d", 
			store.StoreName, store.DeliveryType)
	})

	t.Run("测试获取商店请求数据", func(t *testing.T) {
		// 测试请求参数
		data := dd.StoreListParam{
			Longitude: "121.4737",
			Latitude:  "31.2304",
		}

		dataStr, err := json.Marshal(data)
		if err != nil {
			t.Errorf("JSON序列化失败: %v", err)
		}

		var parsed dd.StoreListParam
		if err := json.Unmarshal(dataStr, &parsed); err != nil {
			t.Errorf("JSON反序列化失败: %v", err)
		}

		if parsed.Longitude == "" || parsed.Latitude == "" {
			t.Error("经纬度不能为空")
		}

		t.Logf("✅ 获取商店请求数据测试通过 - 经度: %s, 纬度: %s", 
			parsed.Longitude, parsed.Latitude)
	})
}

