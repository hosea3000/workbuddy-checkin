#ifndef WORKBUDDY_TRAY_DARWIN_H
#define WORKBUDDY_TRAY_DARWIN_H

// woTrayStart 在 macOS 主线程创建菜单栏图标与「打开 / 退出」菜单。
// iconBytes 为模板图 PNG 字节（纯黑 + alpha），可为空。
void woTrayStart(void *iconBytes, int length);

// woTraySetTip 更新菜单栏 tooltip 文本。
void woTraySetTip(const char *tip);

// woSetAccessoryPolicy 将应用激活策略设为 Accessory（不显示 Dock 图标）。
void woSetAccessoryPolicy(void);

#endif
