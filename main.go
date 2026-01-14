package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
)

type Config struct {
	Debug           bool   `json:"debug"`             // 调试模式
	AutoConnect     bool   `json:"auto_connect"`      // 是否开启自动连接
	AutoConnectIP   string `json:"auto_connect_ip"`   // 自动连接的 IP
	AutoConnectPort string `json:"auto_connect_port"` // 自动连接的端口
	AutoCopy        bool   `json:"auto_copy"`         // 出结果后自动复制
}

var config = Config{
	Debug:           false,
	AutoConnect:     false,       // 默认不启用自动连接
	AutoConnectIP:   "127.0.0.1", // 默认自动连接 IP
	AutoConnectPort: "16384",     // 默认自动连接端口
	AutoCopy:        false,       // 默认关闭
}

// 读取配置文件并加载到 config 变量中
func loadConfig() {
	if file, err := os.Open("config.json"); err == nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&config); err != nil {
			fmt.Println("[ERROR] 读取 config.json 失败:", err)
		}
	}
}

// debugPrint 用于调试打印，只有在 Debug 为 true 时才会打印
func debugPrint(v ...interface{}) {
	if config.Debug {
		fmt.Println(v...)
	}
}

// 等待用户按回车键退出
func waitForExit() {
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// listDevices 用于列出连接的设备
func listDevices() ([]string, error) {
	output, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return nil, err
	}
	lines, devices := strings.Split(string(output), "\n"), []string{}
	for _, line := range lines[1:] {
		if strings.Contains(line, "device") && !strings.Contains(line, "unauthorized") {
			devices = append(devices, strings.Fields(line)[0])
		}
	}
	debugPrint("[DEBUG] 设备列表:", devices) // 打印设备列表（仅在调试模式下）
	return devices, nil
}

// connectToADB 用于连接 ADB 设备
func connectToADB() error {
	output, err := exec.Command("adb", "connect", fmt.Sprintf("%s:%s", config.AutoConnectIP, config.AutoConnectPort)).Output()
	if err != nil || !strings.Contains(string(output), "connected to") {
		return fmt.Errorf("无法连接到设备")
	}
	debugPrint("[DEBUG] adb connect 输出:\n", string(output)) // 打印 adb 连接输出（仅在调试模式下）
	return nil
}

// selectDevice 用于选择设备（如果有多个设备）
func selectDevice(devices []string) (string, error) {
	if len(devices) == 1 {
		return devices[0], nil
	}
	fmt.Println("检测到多个设备，请输入编号选择设备:")
	for i, device := range devices {
		fmt.Printf("[%d] %s\n", i+1, device)
	}
	var choice int
	fmt.Print("输入设备编号 (1 到 ", len(devices), "): ")
	fmt.Scanln(&choice)
	if choice < 1 || choice > len(devices) {
		return "", fmt.Errorf("无效的选择")
	}
	return devices[choice-1], nil
}

// copyToClipboard 用于将文本复制到剪贴板
func copyToClipboard(text string) {
	if !config.AutoCopy {
		return
	}
	if err := clipboard.WriteAll(text); err != nil {
		fmt.Println("[WARN] 复制到剪贴板失败:", err)
		return
	}
	fmt.Println("[INFO] URL 已自动复制到剪贴板")
}

// extractURLs 用于从设备日志中提取 URL
func extractURLs(deviceID string) {
	cmd := exec.Command("adb", "-s", deviceID, "shell", "logcat")
	stdout, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		fmt.Println("[ERROR] 启动 adb logcat 失败:", err)
		waitForExit()
		return
	}
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)

	// 定义正则表达式
	ysURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/hk4e/event/[^\s]+`)                        // 原神
	starRailURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/hkrpg/[^\s]+`)                       // 崩铁
	zzzURLRegex := regexp.MustCompile(`https://webstatic\.mihoyo\.com/nap/event/[^\s]+`)                        // 绝区零
	wuwaURLRegex := regexp.MustCompile(`https://aki-gm-resources\.aki-game\.com/aki/gacha/index\.html#/[^\s]+`) // 鸣潮

	// 从日志中读取每一行并检查是否有符合条件的 URL
	for scanner.Scan() {
		line := scanner.Text()
		if url := ysURLRegex.FindString(line); url != "" {
			fmt.Println("[原神] 找到的URL:", url)
			copyToClipboard(url)
			waitForExit()
			return
		}
		if url := starRailURLRegex.FindString(line); url != "" {
			fmt.Println("[崩坏：星穹铁道] 找到的URL:", url)
			copyToClipboard(url)
			waitForExit()
			return
		}
		if url := zzzURLRegex.FindString(line); url != "" {
			fmt.Println("[绝区零] 找到的URL:", url)
			copyToClipboard(url)
			waitForExit()
			return
		}
		if url := wuwaURLRegex.FindString(line); url != "" {
			fmt.Println("[鸣潮] 找到的URL:", url)
			copyToClipboard(url)
			waitForExit()
			return
		}
	}
	fmt.Println("未找到符合条件的URL")
	waitForExit()
}

func main() {
	fmt.Println("正在检查 ADB 连接状态...")
	loadConfig() // 加载配置文件

	// 第一步：先检查是否有设备连接（使用 adb devices）
	devices, err := listDevices()
	if err != nil || len(devices) == 0 {
		// 第二步：如果未检测到设备且启用了自动连接，则尝试自动连接
		if config.AutoConnect {
			fmt.Printf("未检测到设备，尝试连接到 %s:%s...\n", config.AutoConnectIP, config.AutoConnectPort)
			if err := connectToADB(); err != nil {
				fmt.Println("[ERROR] 无法连接到设备:", err)
				waitForExit()
				return
			}
			// 自动连接后重新检查设备
			devices, err = listDevices()
		}
	}

	// 第三步：如果设备仍未检测到，退出程序
	if err != nil || len(devices) == 0 {
		fmt.Println("仍未检测到设备，程序将退出。")
		waitForExit()
		return
	}

	// 第四步：如果检测到设备，提示用户选择一个设备
	if deviceID, err := selectDevice(devices); err == nil {
		fmt.Println("正在监听日志，请打开抽卡界面...")
		extractURLs(deviceID) // 开始监听日志并提取 URL
	} else {
		fmt.Println("[ERROR] 设备选择错误:", err)
		waitForExit()
	}
}
