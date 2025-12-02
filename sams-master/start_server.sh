#!/bin/bash

echo "===================================="
echo "山姆会员商店自动抢购系统 - Web UI"
echo "===================================="
echo ""
echo "正在启动Web服务器..."
echo ""
echo "服务器将在 http://localhost:8080 启动"
echo "请在浏览器中打开上述地址"
echo ""
echo "按 Ctrl+C 停止服务器"
echo ""

go run . server

