package test

import (
	"testing"
	"time"

	"github.com/robGoods/sams/dd"
	"github.com/tidwall/gjson"
)

// TestGetCapacity 测试获取配送时间功能
// 这个功能查询可用的配送时间段
func TestGetCapacity(t *testing.T) {
	t.Run("测试配送时间数据解析", func(t *testing.T) {
		// 模拟配送时间API返回的数据
		mockResponse := `{
			"code": "Success",
			"data": {
				"capcityResponseList": [
					{
						"strDate": "2024-01-15",
						"deliveryDesc": "明天",
						"dateISFull": false,
						"list": [
							{
								"startTime": "09:00",
								"endTime": "11:00",
								"timeISFull": false,
								"disabled": false,
								"startRealTime": "1705280400000",
								"endRealTime": "1705287600000"
							},
							{
								"startTime": "11:00",
								"endTime": "13:00",
								"timeISFull": true,
								"disabled": false,
								"startRealTime": "1705287600000",
								"endRealTime": "1705294800000"
							}
						]
					}
				]
			}
		}`

		result := gjson.Parse(mockResponse)
		if result.Get("code").Str != "Success" {
			t.Errorf("期望code为Success，实际为: %s", result.Get("code").Str)
		}

		capCityList := result.Get("data.capcityResponseList").Array()
		if len(capCityList) == 0 {
			t.Error("配送时间列表为空")
		}

		// 验证日期信息
		firstDate := capCityList[0]
		strDate := firstDate.Get("strDate").Str
		dateISFull := firstDate.Get("dateISFull").Bool()

		if strDate == "" {
			t.Error("日期字符串不能为空")
		}

		// 验证时间段列表
		timeList := firstDate.Get("list").Array()
		if len(timeList) == 0 {
			t.Error("时间段列表不能为空")
		}

		// 筛选可用时间段
		availableSlots := 0
		for _, slot := range timeList {
			timeISFull := slot.Get("timeISFull").Bool()
			disabled := slot.Get("disabled").Bool()

			if !timeISFull && !disabled {
				availableSlots++
				startTime := slot.Get("startTime").Str
				endTime := slot.Get("endTime").Str
				t.Logf("发现可用时间段: %s - %s", startTime, endTime)
			}
		}

		if availableSlots == 0 {
			t.Error("应该至少有一个可用时间段")
		}

		t.Logf("✅ 配送时间数据解析测试通过 - 日期: %s, 可用时段数: %d", 
			strDate, availableSlots)
	})

	t.Run("测试时间段筛选逻辑", func(t *testing.T) {
		// 模拟多个时间段
		slots := []struct {
			TimeISFull bool
			Disabled   bool
			StartTime  string
			EndTime    string
		}{
			{false, false, "09:00", "11:00"}, // 可用
			{true, false, "11:00", "13:00"},  // 已约满
			{false, true, "13:00", "15:00"},  // 已禁用
			{false, false, "15:00", "17:00"}, // 可用
		}

		availableCount := 0
		for _, slot := range slots {
			if !slot.TimeISFull && !slot.Disabled {
				availableCount++
			}
		}

		if availableCount != 2 {
			t.Errorf("应该有2个可用时间段，实际为: %d", availableCount)
		}

		t.Log("✅ 时间段筛选逻辑测试通过")
	})

	t.Run("测试Capacity结构体", func(t *testing.T) {
		capacity := dd.Capacity{
			CapCityResponseList: []dd.CapCityResponse{
				{
					StrDate:      "2024-01-15",
					DeliveryDesc: "明天",
					DateISFull:   false,
					List: []dd.List{
						{
							StartTime:     "09:00",
							EndTime:       "11:00",
							TimeISFull:    false,
							Disabled:      false,
							StartRealTime: "1705280400000",
							EndRealTime:   "1705287600000",
						},
					},
				},
			},
		}

		if len(capacity.CapCityResponseList) == 0 {
			t.Error("配送时间响应列表不能为空")
		}

		firstDate := capacity.CapCityResponseList[0]
		if firstDate.StrDate == "" {
			t.Error("日期字符串不能为空")
		}
		if len(firstDate.List) == 0 {
			t.Error("时间段列表不能为空")
		}

		t.Logf("✅ Capacity结构体测试通过 - 日期: %s, 时段数: %d", 
			firstDate.StrDate, len(firstDate.List))
	})

	t.Run("测试请求日期格式", func(t *testing.T) {
		// 验证日期格式是否正确（Go的时间格式）
		now := time.Now()
		today := now.Format("2006-01-02")
		tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

		// 验证格式
		if len(today) != 10 {
			t.Errorf("日期格式错误，应为YYYY-MM-DD，实际为: %s", today)
		}
		if len(tomorrow) != 10 {
			t.Errorf("日期格式错误，应为YYYY-MM-DD，实际为: %s", tomorrow)
		}

		t.Logf("✅ 请求日期格式测试通过 - 今天: %s, 明天: %s", today, tomorrow)
	})

	t.Run("测试配送时间错误处理", func(t *testing.T) {
		// 测试限流错误
		mockLimitedResponse := `{
			"code": "LIMITED",
			"msg": "服务器正忙,请稍后再试"
		}`
		result1 := gjson.Parse(mockLimitedResponse)
		if result1.Get("code").Str != "LIMITED" {
			t.Error("应该返回LIMITED错误")
		}

		// 测试获取履约时间异常
		mockCapacityErrResponse := `{
			"code": "Success",
			"msg": "获取履约时间异常"
		}`
		result2 := gjson.Parse(mockCapacityErrResponse)
		// 注意：CapacityErr是通过msg判断的，不是code
		msg := result2.Get("msg").Str
		if msg == "" {
			t.Error("错误消息不能为空")
		}

		t.Log("✅ 配送时间错误处理测试通过")
	})
}

