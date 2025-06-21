# ADB 抽卡记录抓取工具

一个使用 Go 编写的命令行工具，用于从 Android 设备的日志中提取抽卡记录链接，支持以下游戏：

- 原神（Genshin Impact）
- 崩坏：星穹铁道（Honkai: Star Rail）
- 绝区零（Zenless Zone Zero）
- 鸣潮（Wuthering Waves）

## 功能特点

- 自动识别已连接的 ADB 设备；
- 从日志中自动提取抽卡记录 URL；

## 使用说明

### 1. 安装依赖

请先确保你已经安装了以下软件：

- [Go (Golang)](https://golang.org/dl/)
- [ADB (Android Debug Bridge)](https://developer.android.com/studio/releases/platform-tools)

ADB 工具需配置环境变量，确保在命令行中可以直接执行 `adb` 命令。

### 2. 编译

```bash
go build main.go
