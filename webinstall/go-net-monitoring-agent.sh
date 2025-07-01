#!/bin/bash

# go-net-monitoring-agent webinstall package
# https://webinstall.dev/go-net-monitoring-agent

set -e
set -u

pkg_cmd_name="go-net-monitoring-agent"
pkg_dst_cmd="$HOME/.local/bin/agent"
pkg_dst_dir="$HOME/.local/opt/go-net-monitoring"

pkg_get_current_version() {
    echo $(agent --version 2>/dev/null | head -n 1 | cut -d' ' -f2 2>/dev/null || echo "")
}

pkg_install() {
    # 下载并执行安装脚本
    curl -fsSL https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/install.sh | bash -s agent
}

pkg_link() {
    # webinstall.dev 会自动处理链接
    return 0
}

pkg_done_message() {
    echo ""
    echo "🎉 go-net-monitoring-agent 安装完成!"
    echo ""
    echo "📋 使用说明:"
    echo "  配置文件: $pkg_dst_dir/configs/agent.yaml"
    echo "  启动命令: sudo agent --config $pkg_dst_dir/configs/agent.yaml"
    echo "  查看帮助: agent --help"
    echo ""
    echo "⚠️  注意: Agent 需要 root 权限进行网络监控"
    echo ""
}
