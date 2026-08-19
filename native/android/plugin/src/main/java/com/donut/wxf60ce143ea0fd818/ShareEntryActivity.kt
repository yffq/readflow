package com.donut.wxf60ce143ea0fd818

import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import java.net.URLEncoder

/**
 * 系统分享入口（透明 Activity）。
 *
 * 在 AndroidManifest 中通过 ACTION_SEND + text/plain 的 intent-filter 声明，
 * 当用户在其他 App（浏览器、微信等）里把文本分享到 Readflow 时，
 * 系统会启动本 Activity，并把分享文本放在 Intent.EXTRA_TEXT 里。
 *
 * 本 Activity 提取出其中的 URL，重新构造成 readflow:// 深链交给主 App，
 * 小程序侧在 App.onShow(options) 里读取 query.url 后走保存流程。
 */
class ShareEntryActivity : Activity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        routeShare()
        finish()
    }

    private fun routeShare() {
        val url = extractUrl(intent) ?: return
        val deepLink = "readflow://pages/save/save?url=" + URLEncoder.encode(url, "UTF-8")
        val target = Intent(Intent.ACTION_VIEW, Uri.parse(deepLink))
        target.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        startActivity(target)
    }

    private fun extractUrl(intent: Intent): String? {
        if (intent.action != Intent.ACTION_SEND) return null
        val text = intent.getStringExtra(Intent.EXTRA_TEXT)?.trim() ?: return null
        // 分享文本里可能夹着描述，先尝试提取第一个 http(s) 链接
        Regex("https?://[^\\s\"'<>]+").find(text)?.let { return it.value }
        return if (text.startsWith("http://") || text.startsWith("https://")) text else null
    }
}
