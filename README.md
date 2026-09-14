# workbuddy-checkin

一个 Windows 桌面小工具：管理 CodeBuddy（腾讯，`copilot.tencent.com`）账号并**自动完成每日签到**，常驻系统托盘，电脑没开时开机自动补签。

无需服务器、无需数据库、无需开端口——数据只存在本机。提供两种形态：

- **安装版**：中文安装向导（per-user，无需管理员），自动创建快捷方式与卸载项，支持应用内一键检查更新
- **绿色版**：单文件 exe，免安装，双击即用

## 功能特性

- **自动签到**：常驻托盘，每小时巡检一次；应用启动、系统唤醒/解锁时立即补签，当天没签上绝不放过
- **多账号**：逐个签到，账号间随机间隔 5~20 秒，降低上游风控风险
- **OAuth 登录**：点击「添加账号」自动拉起浏览器完成授权，令牌仅存本机（0600 权限）
- **积分可见**：账号卡片实时展示今日签到状态与积分余额，可手动刷新
- **凭证自愈**：临期自动续期；失效时标记「需重新登录」并通知，不会静默失败
- **零配置**：没有需要理解的配置项，装上、登录、然后什么都不用管

## 软件截图

![workbuddy-checkin 主界面](docs/assets/workbuddy-checkin.png)

## 下载安装

前往 [Releases](https://github.com/hosea3000/workbuddy-checkin/releases/latest) 下载：

| 文件 | 说明 |
|---|---|
| `workbuddy-checkin-amd64-installer.exe` | 安装版，中文向导，无需管理员权限 |
| `workbuddy-checkin.exe` | 绿色版单文件，双击即用 |

系统要求：Windows 10 1809+ / Windows 11，x64。

## 使用

1. 启动应用，点击「添加账号」→ 自动打开浏览器完成 CodeBuddy 登录 → 卡片出现新账号
2. 关闭主窗口即最小化到托盘，应用仍在后台调度签到
3. 签到结果通过系统通知告知；打开主界面可查看每个账号的今日状态与积分余额
4. 如需退出，右键托盘图标选择「退出」（关闭窗口不会退出应用）

数据目录：`%APPDATA%\workbuddy-checkin\`，可在设置页点「打开数据目录」直达。

## 隐私与安全

- 所有数据仅保存在本机，不上传任何第三方
- 令牌明文存储且权限为 0600（仅当前用户可读）
- 日志、界面、通知中**永不出现完整令牌**，界面只显示令牌末 8 位
- 网络仅访问 `copilot.tencent.com` 与 GitHub（更新检查/下载，可关闭）

## 从源码构建

前置：Go 1.26、Node.js 22、[Wails CLI v2](https://wails.io/)（构建安装器还需 NSIS）。

```bash
# 1. 先构建前端：main.go 用 //go:embed all:frontend/dist，dist 被 gitignore，
#    全新 clone 若不先建前端，go build/test 会直接失败
cd frontend && npm install && npm run build && cd ..

# 2. 静态检查与测试
go vet ./... && go test ./...

# 3. 构建 Windows 产物
wails build -platform windows/amd64          # 绿色版 exe
wails build -nsis -installscope user -platform windows/amd64  # 安装版

# 开发模式（热重载）
wails dev
```

发布：推送 `v*` tag，`.github/workflows/release.yml` 会自动构建 Windows exe 与 NSIS 安装器并创建 Release。

## 免责声明

本项目仅供**本人授权的 CodeBuddy 账号在本机使用**。请遵守 CodeBuddy 的服务条款，使用产生的任何后果由使用者自行承担。
