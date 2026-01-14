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
	Debug           bool   `json:"debug"`
	AutoConnect     bool   `json:"auto_connect"`
	AutoConnectIP   string `json:"auto_connect_ip"`
	AutoConnectPort string `json:"auto_connect_port"`
	AutoCopy        bool   `json:"auto_copy"`
}

type DeviceInfo struct {
	ID    string
	Model string
}

type URLPattern struct {
	Name  string
	Regex *regexp.Regexp
}

var config = Config{
	Debug:           false,
	AutoConnect:     false,
	AutoConnectIP:   "127.0.0.1",
	AutoConnectPort: "16384",
	AutoCopy:        true,
}

var urlPatterns = []URLPattern{
	{"原神", regexp.MustCompile(`https://webstatic\.mihoyo\.com/hk4e/event/[^\s]+`)},
	{"崩坏：星穹铁道", regexp.MustCompile(`https://webstatic\.mihoyo\.com/hkrpg/[^\s]+`)},
	{"绝区零", regexp.MustCompile(`https://webstatic\.mihoyo\.com/nap/event/[^\s]+`)},
	{"鸣潮", regexp.MustCompile(`https://aki-gm-resources\.aki-game\.com/aki/gacha/index\.html#/[^\s]+`)},
}

// 加载配置文件
func loadConfig() {
	if file, err := os.Open("config.json"); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&config)
	}
}

// 调试打印
func debugPrint(v ...interface{}) {
	if config.Debug {
		fmt.Println(v...)
	}
}

// 等待退出
func waitForExit() {
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// 列出设备
func listDevices() ([]string, error) {
	output, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return nil, err
	}

	var devices []string
	for _, line := range strings.Split(string(output), "\n")[1:] {
		if strings.Contains(line, "device") && !strings.Contains(line, "unauthorized") {
			devices = append(devices, strings.Fields(line)[0])
		}
	}
	debugPrint("[DEBUG] 设备列表:", devices)
	return devices, nil
}

// 获取设备型号
func getDeviceModel(deviceID string) string {
	output, err := exec.Command("adb", "-s", deviceID, "shell", "getprop", "ro.product.model").Output()
	if err != nil {
		debugPrint("[DEBUG] 获取设备型号失败:", deviceID, err)
		return "Unknown Device"
	}
	if model := strings.TrimSpace(string(output)); model != "" {
		return model
	}
	return "Unknown Device"
}

// 获取设备信息列表
func getDevicesWithModel(deviceIDs []string) []DeviceInfo {
	devices := make([]DeviceInfo, len(deviceIDs))
	for i, id := range deviceIDs {
		devices[i] = DeviceInfo{ID: id, Model: getDeviceModel(id)}
	}
	return devices
}

// 连接 ADB
func connectToADB() error {
	output, err := exec.Command("adb", "connect", fmt.Sprintf("%s:%s", config.AutoConnectIP, config.AutoConnectPort)).Output()
	if err != nil || !strings.Contains(string(output), "connected to") {
		return fmt.Errorf("无法连接到设备")
	}
	debugPrint("[DEBUG] adb connect 输出:\n", string(output))
	return nil
}

// 选择设备
func selectDevice(devices []DeviceInfo) (DeviceInfo, error) {
	if len(devices) == 1 {
		return devices[0], nil
	}

	fmt.Println("检测到多个设备，请输入编号选择设备:")
	for i, device := range devices {
		if config.Debug {
			fmt.Printf("[%d] %s (%s)\n", i+1, device.Model, device.ID)
		} else {
			fmt.Printf("[%d] %s\n", i+1, device.Model)
		}
	}

	var choice int
	fmt.Print("输入设备编号 (1 到 ", len(devices), "): ")
	fmt.Scanln(&choice)

	if choice < 1 || choice > len(devices) {
		return DeviceInfo{}, fmt.Errorf("无效的选择")
	}
	return devices[choice-1], nil
}

// 复制到剪贴板
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

// 处理找到的 URL
func handleURLFound(gameName, url string) {
	fmt.Printf("[%s] 找到的URL: %s\n", gameName, url)
	copyToClipboard(url)
	waitForExit()
}

// 提取 URL
func extractURLs(device DeviceInfo) {
	cmd := exec.Command("adb", "-s", device.ID, "shell", "logcat")
	stdout, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		fmt.Println("[ERROR] 启动 adb logcat 失败:", err)
		waitForExit()
		return
	}
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, cap(buf))

	for scanner.Scan() {
		line := scanner.Text()
		// debugPrint("[DEBUG] 日志行:", line)

		for _, pattern := range urlPatterns {
			if url := pattern.Regex.FindString(line); url != "" {
				handleURLFound(pattern.Name, url)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("[ERROR] 读取日志时出错:", err)
	} else {
		fmt.Println("未找到符合条件的URL")
	}
	waitForExit()
}

func main() {
	fmt.Println("正在检查 ADB 连接状态...")
	loadConfig()

	// 检查设备连接
	deviceIDs, err := listDevices()
	if err != nil || len(deviceIDs) == 0 {
		// 尝试自动连接
		if config.AutoConnect {
			fmt.Printf("未检测到设备，尝试连接到 %s:%s...\n", config.AutoConnectIP, config.AutoConnectPort)
			if err := connectToADB(); err != nil {
				fmt.Println("[ERROR] 无法连接到设备:", err)
				waitForExit()
				return
			}
			deviceIDs, err = listDevices()
		}
	}

	// 检查是否有可用设备
	if err != nil || len(deviceIDs) == 0 {
		fmt.Println("仍未检测到设备，程序将退出。")
		waitForExit()
		return
	}

	// 获取设备信息并选择
	devices := getDevicesWithModel(deviceIDs)
	device, err := selectDevice(devices)
	if err != nil {
		fmt.Println("[ERROR] 设备选择错误:", err)
		waitForExit()
		return
	}

	fmt.Printf("正在监听日志（当前设备：%s），请打开抽卡界面...\n", device.Model)
	extractURLs(device)
}
