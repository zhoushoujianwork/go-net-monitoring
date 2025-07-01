#!/bin/bash

# go-net-monitoring-server webinstall package
# https://webinstall.dev/go-net-monitoring-server

set -e
set -u

pkg_cmd_name="go-net-monitoring-server"
pkg_dst_cmd="$HOME/.local/bin/server"
pkg_dst_dir="$HOME/.local/opt/go-net-monitoring"

pkg_get_current_version() {
    echo $(server --version 2>/dev/null | head -n 1 | cut -d' ' -f2 2>/dev/null || echo "")
}

pkg_install() {
    # 下载并执行安装脚本
    curl -fsSL https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/install.sh | bash -s server
}

pkg_link() {
    # webinstall.dev 会自动处理链接
    return 0
}

pkg_done_message() {
    echo ""
    echo "🎉 go-net-monitoring-server 安装完成!"
    echo ""
    echo "📋 使用说明:"
    echo "  配置文件: $pkg_dst_dir/configs/server.yaml"
    echo "  启动命令: server --config $pkg_dst_dir/configs/server.yaml"
    echo "  查看指标: curl http://localhost:8080/metrics"
    echo "  查看帮助: server --help"
    echo ""
}
