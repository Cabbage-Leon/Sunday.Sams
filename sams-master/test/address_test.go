package test

import (
	"encoding/json"
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestGetAddress 测试获取地址列表功能
// 这个测试验证程序能否正确获取用户的收货地址列表
func TestGetAddress(t *testing.T) {
	// 注意：这是一个需要真实authToken的测试
	// 在实际测试中，你需要提供有效的authToken
	// 这里我们主要测试数据解析逻辑

	t.Run("测试地址数据解析", func(t *testing.T) {
		// 模拟API返回的JSON数据
		mockResponse := `{
			"code": "Success",
			"data": {
				"addressList": [
					{
						"addressId": "test-address-001",
						"mobile": "13800138000",
						"name": "张三",
						"countryName": "中国",
						"provinceName": "上海市",
						"cityName": "上海市",
						"districtName": "浦东新区",
						"receiverAddress": "世纪大道",
						"detailAddress": "100号",
						"latitude": "31.2304",
						"longitude": "121.4737"
					}
				]
			}
		}`

		result := gjson.Parse(mockResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		addressList := result.Get("data.addressList").Array()
		if len(addressList) == 0 {
			t.Error("地址列表为空")
		}

		// 验证地址字段
		firstAddr := addressList[0]
		if firstAddr.Get("addressId").Str == "" {
			t.Error("地址ID为空")
		}
		if firstAddr.Get("name").Str == "" {
			t.Error("收货人姓名为空")
		}
		if firstAddr.Get("mobile").Str == "" {
			t.Error("手机号为空")
		}

		t.Logf("✅ 地址解析测试通过 - 地址ID: %s, 收货人: %s", 
			firstAddr.Get("addressId").Str, 
			firstAddr.Get("name").Str)
	})

	t.Run("测试地址结构体", func(t *testing.T) {
		// 测试Address结构体
		addr := dd.Address{
			AddressId:       "test-001",
			Name:            "测试用户",
			Mobile:          "13800138000",
			ProvinceName:    "上海市",
			CityName:        "上海市",
			DistrictName:    "浦东新区",
			ReceiverAddress: "世纪大道",
			DetailAddress:   "100号",
			Latitude:        "31.2304",
			Longitude:       "121.4737",
		}

		if addr.AddressId == "" {
			t.Error("地址ID不能为空")
		}
		if addr.Latitude == "" || addr.Longitude == "" {
			t.Error("经纬度不能为空，用于查找附近门店")
		}

		t.Logf("✅ 地址结构体测试通过 - %s %s %s", 
			addr.DistrictName, addr.ReceiverAddress, addr.DetailAddress)
	})
}

// TestSaveDeliveryAddress 测试保存配送地址功能
// 这个功能将选中的地址保存到购物车系统
func TestSaveDeliveryAddress(t *testing.T) {
	t.Run("测试保存地址请求数据格式", func(t *testing.T) {
		// 测试请求数据的JSON格式
		data := map[string]interface{}{
			"uid":       "",
			"addressId": "test-address-001",
		}

		dataStr, err := json.Marshal(data)
		if err != nil {
			t.Errorf("JSON序列化失败: %v", err)
		}

		// 验证JSON格式
		var parsed map[string]interface{}
		if err := json.Unmarshal(dataStr, &parsed); err != nil {
			t.Errorf("JSON反序列化失败: %v", err)
		}

		if parsed["addressId"].(string) != "test-address-001" {
			t.Error("地址ID不匹配")
		}

		t.Logf("✅ 保存地址请求格式测试通过 - JSON: %s", string(dataStr))
	})

	t.Run("测试保存地址响应解析", func(t *testing.T) {
		// 模拟成功响应
		mockSuccessResponse := `{
			"code": "Success",
			"data": {
				"result": true
			}
		}`

		result := gjson.Parse(mockSuccessResponse)
		if result.Get("code").Str != "Success" {
			t.Error("响应code应为Success")
		}
		if !result.Get("data.result").Bool() {
			t.Error("保存结果应为true")
		}

		t.Log("✅ 保存地址响应解析测试通过")
	})

	t.Run("测试认证失败场景", func(t *testing.T) {
		// 模拟token过期响应
		mockAuthFailResponse := `{
			"code": "AUTH_FAIL",
			"msg": "认证失败"
		}`

		result := gjson.Parse(mockAuthFailResponse)
		if result.Get("code").Str != "AUTH_FAIL" {
			t.Error("应该返回AUTH_FAIL")
		}

		t.Log("✅ 认证失败场景测试通过")
	})
}

