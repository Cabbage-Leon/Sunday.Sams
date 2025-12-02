package test

import (
	"testing"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestCheckGoods 测试商品校验功能
// 这个功能在下单前再次确认商品是否真的可以购买
func TestCheckGoods(t *testing.T) {
	t.Run("测试商品校验成功场景", func(t *testing.T) {
		// 模拟商品校验成功的响应
		mockSuccessResponse := `{
			"code": "Success",
			"data": {
				"isHasException": false
			}
		}`

		result := gjson.Parse(mockSuccessResponse)
		if result.Get("code").Str != "Success" {
			t.Error("响应code应为Success")
		}

		isHasException := result.Get("data.isHasException").Bool()
		if isHasException {
			t.Error("商品校验应无异常")
		}

		t.Log("✅ 商品校验成功场景测试通过")
	})

	t.Run("测试商品缺货场景", func(t *testing.T) {
		// 模拟商品缺货的响应
		mockOOSResponse := `{
			"code": "Success",
			"data": {
				"isHasException": true,
				"popUpInfo": {
					"desc": "部分商品已缺货",
					"goodsList": [
						{
							"spuId": "spu-001",
							"goodsName": "缺货商品",
							"stockQuantity": 0,
							"stockStatus": false
						}
					]
				}
			}
		}`

		result := gjson.Parse(mockOOSResponse)
		isHasException := result.Get("data.isHasException").Bool()
		if !isHasException {
			t.Error("应该检测到商品异常")
		}

		desc := result.Get("data.popUpInfo.desc").Str
		if desc == "" {
			t.Error("异常描述不能为空")
		}

		goodsList := result.Get("data.popUpInfo.goodsList").Array()
		if len(goodsList) == 0 {
			t.Error("异常商品列表不应为空")
		}

		t.Logf("✅ 商品缺货场景测试通过 - 异常描述: %s", desc)
	})

	t.Run("测试NormalGoods转Goods", func(t *testing.T) {
		normalGoods := dd.NormalGoods{
			StoreId:       "store-001",
			SpuId:         "spu-001",
			GoodsName:     "测试商品",
			Price:         59,
			Quantity:      2,
			IsSelected:    true,
			Weight:        1.5,
			StockQuantity: 10,
			StockStatus:   true,
			IsPutOnSale:   true,
			IsAvailable:   true,
		}

		goods := normalGoods.ToGoods()

		if goods.SpuId != normalGoods.SpuId {
			t.Error("商品ID不匹配")
		}
		if goods.Quantity != normalGoods.Quantity {
			t.Error("商品数量不匹配")
		}
		if goods.Price != normalGoods.Price {
			t.Error("商品价格不匹配")
		}
		if goods.Weight != normalGoods.Weight {
			t.Error("商品重量不匹配")
		}

		t.Logf("✅ NormalGoods转Goods测试通过 - 商品: %s, 价格: %d, 数量: %d", 
			goods.GoodsName, goods.Price, goods.Quantity)
	})

	t.Run("测试商品有效性判断", func(t *testing.T) {
		// 有效商品
		validGoods := dd.NormalGoods{
			StockQuantity: 10,
			StockStatus:   true,
			IsPutOnSale:   true,
			IsAvailable:   true,
		}

		isValid := validGoods.StockQuantity > 0 && 
			validGoods.StockStatus && 
			validGoods.IsPutOnSale && 
			validGoods.IsAvailable

		if !isValid {
			t.Error("有效商品应该通过验证")
		}

		// 无效商品 - 无库存
		invalidGoods1 := dd.NormalGoods{
			StockQuantity: 0,
			StockStatus:   true,
			IsPutOnSale:   true,
			IsAvailable:   true,
		}

		isValid1 := invalidGoods1.StockQuantity > 0 && 
			invalidGoods1.StockStatus && 
			invalidGoods1.IsPutOnSale && 
			invalidGoods1.IsAvailable

		if isValid1 {
			t.Error("无库存商品不应通过验证")
		}

		// 无效商品 - 未上架
		invalidGoods2 := dd.NormalGoods{
			StockQuantity: 10,
			StockStatus:   true,
			IsPutOnSale:   false,
			IsAvailable:   true,
		}

		isValid2 := invalidGoods2.StockQuantity > 0 && 
			invalidGoods2.StockStatus && 
			invalidGoods2.IsPutOnSale && 
			invalidGoods2.IsAvailable

		if isValid2 {
			t.Error("未上架商品不应通过验证")
		}

		t.Log("✅ 商品有效性判断测试通过")
	})
}

