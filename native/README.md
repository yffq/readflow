# 系统级分享（分享到 Readflow）

目标：在系统分享面板里选中 Readflow 时，把分享内容里的 URL 存入 Readflow，效果等同手动「+ Save」。

## 整体架构

```
系统分享面板
   │  携带 URL（text/plain）
   ▼
原生入口（各端不同）
   │  提取 URL，构造成 readflow://pages/save/save?url=<编码后URL>
   ▼
框架深链通道（readflow:// scheme）
   ▼
小程序 App.onShow(options)
   │  utils/share.js 提取 query.url
   ▼
pages/save/save?url=...（与手动 + Save 一致）
```

小程序侧（`miniprogram/utils/share.js` + `app.js`）已实现并通过单元测试，
与平台无关，是三端共用的一环。

## 平台现状

| 平台 | 状态 | 说明 |
|------|------|------|
| Android | ✅ 代码已提供 | 原生插件加 ACTION_SEND，见下文 |
| iOS | ⚠️ 受阻 | 系统分享面板需要 Share Extension（独立 target），当前 v2 插件机制不支持 |
| 鸿蒙 | ⚠️ 受阻 | 当前 DevTools 版本的 ohos 工具链无原生插件能力 |

## Android

### 已提供文件（native/android-share/）

```
plugin/src/main/AndroidManifest.xml                     # ACTION_SEND + text/plain intent-filter
plugin/src/main/java/com/readflow/share/ReadflowSharePlugin.kt   # 插件入口（注册到运行时）
plugin/src/main/java/com/readflow/share/ShareEntryActivity.kt    # 透明 Activity，接收分享并转深链
plugin/src/main/resources/META-INF/services/
    com.tencent.luggage.wxa.SaaA.plugin.NativePluginInterface      # 服务发现（指向 ReadflowSharePlugin）
```

### 接入步骤

1. 在微信开发者工具里用「原生模块」功能创建一个 Android 原生模块（会生成模板工程）。
2. 把上述 4 个文件按相同相对路径放进生成的原生模块工程（包名统一为 `com.readflow.share`，若改名需同步修改所有文件里的包名）。
3. 在原生模块工程里构建出 AAR（`com.donut.plugin:<pluginId>:<pluginVersion>`）。
4. 在开发者工具的「多端应用 → 原生模块」里勾选该插件（会自动写入 `project.miniapp.json` 的 `mini-android.nativePlugins`）。
5. `project.miniapp.json` 里已配置：
   ```json
   "mini-android": {
     "schemes": { "scheme": "readflow", "appLink": [] }
   }
   ```
   这会在主 Activity 上注册 `readflow://` 深链，供 ShareEntryActivity 转跳。
6. 打包安装到 Android 真机，从浏览器/微信里「分享 → Readflow」验证。

### 需注意

- 插件 AAR 的 `pluginId` 与 `ReadflowSharePlugin.getPluginID()`（`BuildConfig.PLUGIN_ID`）
  以及 `project.miniapp.json` 里的 `nativePlugins[].pluginId` 三者必须一致。
- 分享文本若不是 URL（纯文字），当前实现会直接忽略；如需支持「把任意文本当标题」可后续扩展。

## iOS（受阻，待决策）

系统分享面板（UIActivityViewController）只会展示注册了 Share Extension 的 App，
这需要一个独立的 Xcode target：

- `NSExtensionPointIdentifier = com.apple.share-services`
- `NSExtensionActivationRule` 接收 `public.url` / `public.text`
- ShareViewController 读取 URL 后 `openURL(readflow://pages/save/save?url=...)`

但当前 Donut v2 的原生插件只能产出框架（framework），构建系统里唯一内置的
扩展是 TPNS 推送扩展，没有「添加自定义 App Extension」的通用能力。
要落地 iOS 分享，只能切换到**自定义原生工程**（v1 的 useProjectTemplate 模式），
在 Xcode 工程里手加 Share Extension target，改动较大。

## 鸿蒙（受阻，待决策）

当前 DevTools 的 ohos 工具链没有 createNativePlugin / 原生插件类型定义，
同样需要自定义原生鸿蒙工程来注册分享 Want 处理。

## 测试计划

### 已自动化
- `node --test miniprogram/test/share.test.js`：分享 URL 提取、去重、跳转路径、设置引导等 12 个单测。

### 待真机验证（原生构建后）
1. 正常路径：浏览器分享一个 https 链接 → 选择 Readflow → 自动进入保存页且 URL 已填充。
2. 边界：分享纯文字（无 URL）→ 不应崩溃、不进入保存页。
3. 边界：分享文本夹带多个 URL → 取第一个。
4. 异常：未配置 apiUrl/apiKey 时分享 → 先进入设置页，设置完成后回到保存页。
5. 异常：已配置时重复分享同一链接 → 只处理一次（去重生效）。
6. 异常：冷启动（App 未运行）直接分享拉起 → 能正确进入保存流程。
