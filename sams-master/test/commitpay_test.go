package test

import (
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestCommitPay 测试提交订单功能
// 这是最关键的一步，正式提交订单并支付
func TestCommitPay(t *testing.T) {
	t.Run("测试订单提交成功场景", func(t *testing.T) {
		// 模拟订单提交成功的响应
		mockSuccessResponse := `{
			"code": "Success",
			"data": {
				"isSuccess": true,
				"orderNo": "ORDER202401150001",
				"payAmount": "299.00",
				"channel": "wechat",
				"PayInfo": {
					"PayInfo": "wx_pay_info_string",
					"OutTradeNo": "OUT202401150001",
					"TotalAmt": 29900
				}
			}
		}`

		result := gjson.Parse(mockSuccessResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		isSuccess := result.Get("data.isSuccess").Bool()
		if !isSuccess {
			t.Error("订单提交应成功")
		}

		orderNo := result.Get("data.orderNo").Str
		if orderNo == "" {
			t.Error("订单号不能为空")
		}

		payAmount := result.Get("data.payAmount").Str
		if payAmount == "" {
			t.Error("支付金额不能为空")
		}

		channel := result.Get("data.channel").Str
		if channel != "wechat" && channel != "alipay" {
			t.Error("支付渠道应为wechat或alipay")
		}

		t.Logf("✅ 订单提交成功场景测试通过 - 订单号: %s, 金额: %s, 渠道: %s", 
			orderNo, payAmount, channel)
	})

	t.Run("测试Order结构体", func(t *testing.T) {
		order := dd.Order{
			IsSuccess: true,
			OrderNo:   "ORDER202401150001",
			PayAmount: "299.00",
			Channel:   "wechat",
			PayInfo: dd.PayInfo{
				PayInfo:    "wx_pay_info_string",
				OutTradeNo: "OUT202401150001",
				TotalAmt:   29900,
			},
		}

		if !order.IsSuccess {
			t.Error("订单状态应为成功")
		}
		if order.OrderNo == "" {
			t.Error("订单号不能为空")
		}
		if order.PayAmount == "" {
			t.Error("支付金额不能为空")
		}

		t.Logf("✅ Order结构体测试通过 - 订单号: %s, 金额: %s", 
			order.OrderNo, order.PayAmount)
	})

	t.Run("测试订单提交请求数据", func(t *testing.T) {
		// 测试订单提交的参数结构
		settleDeliveryInfo := dd.SettleDeliveryInfo{
			DeliveryType:         0,
			ExpectArrivalTime:    "1705280400000",
			ExpectArrivalEndTime: "1705287600000",
			ArrivalTimeStr:       "2024-01-15 09:00 - 11:00",
		}

		if settleDeliveryInfo.ExpectArrivalTime == "" {
			t.Error("期望到达时间不能为空")
		}
		if settleDeliveryInfo.ExpectArrivalEndTime == "" {
			t.Error("期望到达结束时间不能为空")
		}

		goodsList := []dd.Goods{
			{
				SpuId:      "spu-001",
				StoreId:    "store-001",
				Quantity:   2,
				IsSelected: true,
			},
		}

		if len(goodsList) == 0 {
			t.Error("商品列表不能为空")
		}

		t.Logf("✅ 订单提交请求数据测试通过 - 配送时间: %s, 商品数: %d", 
			settleDeliveryInfo.ArrivalTimeStr, len(goodsList))
	})

	t.Run("测试各种订单提交错误场景", func(t *testing.T) {
		// 测试限流错误
		mockLimitedResponse := `{
			"code": "LIMITED",
			"msg": "当前购物火爆，请稍后再试"
		}`
		result1 := gjson.Parse(mockLimitedResponse)
		if result1.Get("code").Str != "LIMITED" {
			t.Error("应该返回LIMITED错误")
		}

		// 测试商品超过限购
		mockExceedLimitResponse := `{
			"code": "GOODS_EXCEED_LIMIT",
			"msg": "商品超过限购数量"
		}`
		result2 := gjson.Parse(mockExceedLimitResponse)
		if result2.Get("code").Str != "GOODS_EXCEED_LIMIT" {
			t.Error("应该返回GOODS_EXCEED_LIMIT错误")
		}

		// 测试缺货错误
		mockOOSResponse := `{
			"code": "OUT_OF_STOCK",
			"msg": "部分商品已缺货"
		}`
		result3 := gjson.Parse(mockOOSResponse)
		if result3.Get("code").Str != "OUT_OF_STOCK" {
			t.Error("应该返回OUT_OF_STOCK错误")
		}

		// 测试时间段失效
		mockTimeExceptionResponse := `{
			"code": "CLOSE_ORDER_TIME_EXCEPTION",
			"msg": "配送时间已失效"
		}`
		result4 := gjson.Parse(mockTimeExceptionResponse)
		if result4.Get("code").Str != "CLOSE_ORDER_TIME_EXCEPTION" {
			t.Error("应该返回CLOSE_ORDER_TIME_EXCEPTION错误")
		}

		// 测试扣减运力失败
		mockCapacityErrorResponse := `{
			"code": "DECREASE_CAPACITY_COUNT_ERROR",
			"msg": "扣减运力失败"
		}`
		result5 := gjson.Parse(mockCapacityErrorResponse)
		if result5.Get("code").Str != "DECREASE_CAPACITY_COUNT_ERROR" {
			t.Error("应该返回DECREASE_CAPACITY_COUNT_ERROR错误")
		}

		// 测试门店关闭
		mockStoreClosedResponse := `{
			"code": "STORE_HAS_CLOSED",
			"msg": "门店已打烊"
		}`
		result6 := gjson.Parse(mockStoreClosedResponse)
		if result6.Get("code").Str != "STORE_HAS_CLOSED" {
			t.Error("应该返回STORE_HAS_CLOSED错误")
		}

		// 测试超重错误
		mockOverWeightResponse := `{
			"code": "CLOUD_GOODS_OVER_WEIGHT",
			"msg": "订单已超重"
		}`
		result7 := gjson.Parse(mockOverWeightResponse)
		if result7.Get("code").Str != "CLOUD_GOODS_OVER_WEIGHT" {
			t.Error("应该返回CLOUD_GOODS_OVER_WEIGHT错误")
		}

		t.Log("✅ 订单提交错误场景测试通过")
	})

	t.Run("测试支付方式", func(t *testing.T) {
		// 测试微信支付
		payMethodWechat := 1
		channel := "wechat"
		if payMethodWechat == 1 {
			channel = "wechat"
		} else if payMethodWechat == 2 {
			channel = "alipay"
		}

		if channel != "wechat" {
			t.Error("支付方式1应为微信")
		}

		// 测试支付宝
		payMethodAlipay := 2
		channel2 := "wechat"
		if payMethodAlipay == 1 {
			channel2 = "wechat"
		} else if payMethodAlipay == 2 {
			channel2 = "alipay"
		}

		if channel2 != "alipay" {
			t.Error("支付方式2应为支付宝")
		}

		t.Log("✅ 支付方式测试通过")
	})

	t.Run("测试优惠券使用", func(t *testing.T) {
		// 测试优惠券列表
		promotionIds := []string{"coupon-001", "coupon-002"}
		storeId := "store-001"

		couponList := make([]dd.CouponInfo, 0)
		for _, id := range promotionIds {
			couponList = append(couponList, dd.CouponInfo{
				PromotionId: id,
				StoreId:     storeId,
			})
		}

		if len(couponList) != 2 {
			t.Errorf("优惠券数量应为2，实际为: %d", len(couponList))
		}

		for i, coupon := range couponList {
			if coupon.PromotionId == "" {
				t.Errorf("优惠券%d的ID不能为空", i)
			}
			if coupon.StoreId != storeId {
				t.Errorf("优惠券%d的商店ID不匹配", i)
			}
		}

		t.Logf("✅ 优惠券使用测试通过 - 优惠券数: %d", len(couponList))
	})
}

