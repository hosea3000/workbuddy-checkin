//go:build darwin

// tray_darwin.m — macOS 菜单栏（NSStatusItem）实现。
//
// 设计要点：
//   - 完全不碰 NSApplication.delegate，避免与 Wails 的 AppDelegate 冲突。
//   - 所有 Cocoa 操作经 dispatch_async 投递到主队列（Go 侧调用可能来自任意 goroutine）。
//   - 左键单击直接弹出菜单，不依赖 delegate 回调。
//   - 图标用模板图（image.template = YES），自动适配深浅色菜单栏。

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>
#import "tray_darwin.h"

// 由 Go 侧 //export 导出的回调。
extern void woTrayOpen(void);
extern void woTrayQuit(void);

@interface WOTrayController : NSObject
@property (strong) NSStatusItem *statusItem;
@property (strong) NSMenu *menu;
@end

@implementation WOTrayController

- (instancetype)init {
    self = [super init];
    if (self) {
        _statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        _menu = [[NSMenu alloc] init];
        [_menu setAutoenablesItems:NO];

        NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"打开"
                                                          action:@selector(onOpen:)
                                                   keyEquivalent:@""];
        [openItem setTarget:self];
        [_menu addItem:openItem];

        [_menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"退出"
                                                          action:@selector(onQuit:)
                                                   keyEquivalent:@""];
        [quitItem setTarget:self];
        [_menu addItem:quitItem];

        // 左键单击弹出菜单（菜单栏惯例），不设置 statusItem.menu，避免自动弹出。
        NSStatusBarButton *button = _statusItem.button;
        [button setTarget:self];
        [button setAction:@selector(onClick:)];
        [button sendActionOn:(NSEventMaskLeftMouseUp)];
    }
    return self;
}

- (void)onClick:(id)sender {
    [_statusItem popUpStatusItemMenu:_menu];
}

- (void)onOpen:(id)sender {
    woTrayOpen();
}

- (void)onQuit:(id)sender {
    woTrayQuit();
}

- (void)setIcon:(NSData *)data {
    if (data == nil) {
        return;
    }
    NSImage *image = [[NSImage alloc] initWithData:data];
    if (image == nil) {
        return;
    }
    [image setSize:NSMakeSize(16, 16)];
    [image setTemplate:YES];
    _statusItem.button.image = image;
}

@end

static WOTrayController *gController = nil;

static void ensureController(void) {
    if (gController == nil) {
        gController = [[WOTrayController alloc] init];
    }
}

void woTrayStart(void *iconBytes, int length) {
    NSData *data = nil;
    if (iconBytes != NULL && length > 0) {
        data = [NSData dataWithBytes:iconBytes length:(NSUInteger)length];
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        ensureController();
        [gController setIcon:data];
    });
}

void woTraySetTip(const char *tip) {
    if (tip == NULL) {
        return;
    }
    NSString *title = [NSString stringWithUTF8String:tip];
    dispatch_async(dispatch_get_main_queue(), ^{
        ensureController();
        // NSStatusItem 无原生 tooltip；写入 button.toolTip 以便悬停提示。
        gController.statusItem.button.toolTip = title;
    });
}

void woSetAccessoryPolicy(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [[NSApplication sharedApplication]
            setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}
