#!/usr/bin/env python3
"""
简单的HTTP服务器，用于生成入站流量测试
"""

import http.server
import socketserver
import threading
import time
import requests
import json

class TestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        
        # 发送一个较大的响应来生成入站流量
        response_data = {
            "message": "This is a test response to generate inbound traffic",
            "timestamp": time.time(),
            "data": "x" * 1000  # 1KB的数据
        }
        
        self.wfile.write(json.dumps(response_data).encode())

def start_server():
    """启动测试服务器"""
    PORT = 8888
    with socketserver.TCPServer(("", PORT), TestHandler) as httpd:
        print(f"测试服务器启动在端口 {PORT}")
        httpd.serve_forever()

def make_requests():
    """发送测试请求"""
    time.sleep(2)  # 等待服务器启动
    
    for i in range(10):
        try:
            response = requests.get("http://localhost:8888/test")
            print(f"请求 {i+1}: 状态码 {response.status_code}, 响应大小 {len(response.content)} 字节")
            time.sleep(1)
        except Exception as e:
            print(f"请求 {i+1} 失败: {e}")

if __name__ == "__main__":
    # 启动服务器线程
    server_thread = threading.Thread(target=start_server, daemon=True)
    server_thread.start()
    
    # 发送测试请求
    make_requests()
