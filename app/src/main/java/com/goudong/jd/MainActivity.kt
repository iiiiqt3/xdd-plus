package com.goudong.jd

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.Menu
import android.view.View
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.viewpager2.adapter.FragmentStateAdapter
import androidx.viewpager2.widget.ViewPager2
import com.goudong.jd.ui.auth.AuthActivity
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.home.HomeFragment
import com.goudong.jd.ui.jd.JdFragment
import com.goudong.jd.ui.more.MoreFragment
import com.goudong.jd.ui.projects.ProjectsFragment
import com.goudong.jd.ui.tasks.TasksFragment
import com.goudong.jd.update.ApkInstaller
import com.goudong.jd.update.UpdateChecker
import com.goudong.jd.update.UpdateInfo
import com.goudong.jd.push.PushManager
import com.google.android.material.bottomnavigation.BottomNavigationView
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {
    private lateinit var bottomNav: BottomNavigationView
    private lateinit var viewPager: ViewPager2
    private var pendingTabId: Int? = null
    private var isSyncing = false

    private val tabOrder = intArrayOf(TAB_HOME, TAB_PROJECTS, TAB_TASKS, TAB_JD, TAB_MORE)

    private val authLauncher = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        AppServices.isAuthInProgress = false
        if (result.resultCode == Activity.RESULT_OK) {
            val target = pendingTabId ?: TAB_HOME
            switchToTab(target)
            reloadVisibleFragment()
        }
        pendingTabId = null
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        title = "狗东"

        window.statusBarColor = ContextCompat.getColor(this, R.color.surface_soft)
        window.decorView.systemUiVisibility = View.SYSTEM_UI_FLAG_LAYOUT_STABLE or View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR

        val statusBarHeight = run {
            val resId = resources.getIdentifier("status_bar_height", "dimen", "android")
            if (resId > 0) resources.getDimensionPixelSize(resId) else 0
        }

        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(ContextCompat.getColor(this@MainActivity, R.color.surface_soft))
            fitsSystemWindows = true
            setPadding(0, statusBarHeight, 0, 0)
        }

        viewPager = ViewPager2(this).apply {
            id = View.generateViewId()
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, 0, 1f)
            adapter = MainPagerAdapter(this@MainActivity)
            isUserInputEnabled = true
            offscreenPageLimit = 4
            registerOnPageChangeCallback(object : ViewPager2.OnPageChangeCallback() {
                override fun onPageSelected(position: Int) {
                    if (isSyncing) return
                    val tabId = tabOrder[position]
                    if (requiresAuth(tabId) && !AppServices.sessionManager.isAuthenticated()) {
                        pendingTabId = tabId
                        isSyncing = true
                        viewPager.setCurrentItem(tabOrder.indexOf(TAB_JD), false)
                        isSyncing = false
                        authLauncher.launch(Intent(this@MainActivity, AuthActivity::class.java))
                        return
                    }
                    isSyncing = true
                    bottomNav.selectedItemId = tabId
                    isSyncing = false
                    onTabChanged(tabId)
                }
            })
        }

        bottomNav = BottomNavigationView(this).apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(54)).apply {
                marginStart = dp(14)
                marginEnd = dp(14)
                topMargin = dp(4)
                bottomMargin = dp(10)
            }
            minimumHeight = dp(54)
            itemIconSize = dp(18)
            setPadding(0, dp(1), 0, dp(1))
            setItemPaddingTop(dp(3))
            setItemPaddingBottom(dp(3))
            elevation = dp(10).toFloat()
            setBackgroundColor(Color.TRANSPARENT)
            background = GradientDrawable().apply {
                setColor(ContextCompat.getColor(context, R.color.surface_card))
                cornerRadius = dp(22).toFloat()
                setStroke(dp(1), Color.parseColor("#E2E8F0"))
            }
            itemIconTintList = android.content.res.ColorStateList(
                arrayOf(
                    intArrayOf(android.R.attr.state_checked),
                    intArrayOf()
                ),
                intArrayOf(
                    ContextCompat.getColor(context, R.color.brand_primary),
                    Color.parseColor("#94A3B8")
                )
            )
            itemTextColor = android.content.res.ColorStateList(
                arrayOf(
                    intArrayOf(android.R.attr.state_checked),
                    intArrayOf()
                ),
                intArrayOf(
                    ContextCompat.getColor(context, R.color.brand_primary),
                    Color.parseColor("#94A3B8")
                )
            )
            menu.add(Menu.NONE, TAB_HOME, 0, "首页").setIcon(android.R.drawable.ic_menu_view)
            menu.add(Menu.NONE, TAB_PROJECTS, 1, "项目").setIcon(android.R.drawable.ic_menu_agenda)
            menu.add(Menu.NONE, TAB_TASKS, 2, "任务").setIcon(android.R.drawable.star_big_on)
            menu.add(Menu.NONE, TAB_JD, 3, "京东").setIcon(android.R.drawable.ic_menu_send)
            menu.add(Menu.NONE, TAB_MORE, 4, "更多").setIcon(android.R.drawable.ic_menu_manage)
            setOnItemSelectedListener { item ->
                if (isSyncing) return@setOnItemSelectedListener true
                val position = tabOrder.indexOf(item.itemId)
                if (position >= 0) {
                    if (requiresAuth(item.itemId) && !AppServices.sessionManager.isAuthenticated()) {
                        pendingTabId = item.itemId
                        authLauncher.launch(Intent(this@MainActivity, AuthActivity::class.java))
                        return@setOnItemSelectedListener false
                    }
                    isSyncing = true
                    viewPager.setCurrentItem(position, true)
                    isSyncing = false
                    onTabChanged(item.itemId)
                }
                true
            }
        }

        root.addView(viewPager)
        root.addView(bottomNav)
        setContentView(root)

        if (savedInstanceState == null) {
            val defaultTab = if (AppServices.sessionManager.isAuthenticated()) TAB_HOME else TAB_JD
            bottomNav.selectedItemId = defaultTab
            onTabChanged(defaultTab)
        }

        checkUpdate()
    }

    override fun onResume() {
        super.onResume()
        PushManager.onAppForeground()
        if (AppServices.sessionManager.isAuthenticated()) {
            PushManager.startMonitoring(this)
        }
    }

    override fun onPause() {
        super.onPause()
        PushManager.onAppBackground()
    }

    fun updateMoreBadge(unreadCount: Int) {
        runOnUiThread {
            val badge = bottomNav.getOrCreateBadge(TAB_MORE)
            if (unreadCount > 0) {
                badge.isVisible = true
                badge.number = unreadCount.coerceAtMost(99)
                badge.badgeTextColor = Color.WHITE
                badge.backgroundColor = Color.parseColor("#EF4444")
            } else {
                badge.isVisible = false
                badge.clearNumber()
            }
        }
    }

    fun openTab(tabId: Int) {
        switchToTab(tabId)
    }

    private fun switchToTab(tabId: Int) {
        val position = tabOrder.indexOf(tabId)
        if (position < 0) return
        isSyncing = true
        bottomNav.selectedItemId = tabId
        viewPager.setCurrentItem(position, true)
        isSyncing = false
        onTabChanged(tabId)
    }

    private fun onTabChanged(tabId: Int) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        if (tabId == TAB_HOME || tabId == TAB_PROJECTS || tabId == TAB_TASKS) {
            lifecycleScope.launch {
                runCatching { AppServices.portalRepository.verifySession() }
                    .onFailure { error ->
                        val apiError = error as? com.goudong.jd.data.model.ApiError
                        if (apiError?.unauthorized == true) {
                            handleUnauthorized()
                        }
                    }
            }
        }
        updateNotificationBadge()
    }

    private fun updateNotificationBadge() {
        if (!AppServices.sessionManager.isAuthenticated()) return
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchNotifications(includeContent = false) }
                .onSuccess { response -> updateMoreBadge(response.unread) }
        }
    }

    fun handleUnauthorized() {
        if (AppServices.isAuthInProgress) return
        AppServices.isAuthInProgress = true

        lifecycleScope.launch {
            try {
                val credentials = AppServices.sessionManager.loadCredentials()
                if (credentials != null) {
                    val loginResult = runCatching {
                        AppServices.authRepository.login(credentials.first, credentials.second)
                    }
                    if (loginResult.isSuccess) {
                        reloadVisibleFragment()
                        return@launch
                    }
                }

                AppServices.sessionManager.setAuthenticated(false)
                AppServices.apiClient.clearCookies()
                pendingTabId = tabOrder[viewPager.currentItem]
                authLauncher.launch(Intent(this@MainActivity, AuthActivity::class.java))
            } finally {
                AppServices.isAuthInProgress = false
            }
        }
    }

    private fun reloadVisibleFragment() {
        for (fragment in supportFragmentManager.fragments) {
            if (!fragment.isVisible) continue
            when (fragment) {
                is HomeFragment -> fragment.loadData(forceRefresh = true)
                is ProjectsFragment -> fragment.refreshCurrentTab()
                is TasksFragment -> fragment.refreshDashboard()
            }
            break
        }
    }

    private fun requiresAuth(tabId: Int): Boolean {
        return tabId == TAB_HOME || tabId == TAB_PROJECTS || tabId == TAB_MORE || tabId == TAB_TASKS
    }

    private inner class MainPagerAdapter(activity: AppCompatActivity) : FragmentStateAdapter(activity) {
        override fun getItemCount(): Int = tabOrder.size
        override fun createFragment(position: Int): Fragment {
            return when (tabOrder[position]) {
                TAB_HOME -> HomeFragment()
                TAB_PROJECTS -> ProjectsFragment()
                TAB_JD -> JdFragment()
                TAB_TASKS -> TasksFragment()
                TAB_MORE -> MoreFragment()
                else -> JdFragment()
            }
        }
    }

    // ==================== 在线更新 ====================

    private val updateChecker by lazy { UpdateChecker(AppServices.apiClient, this) }

    private fun checkUpdate() {
        lifecycleScope.launch {
            val update = updateChecker.check() ?: return@launch
            showUpdateDialog(update)
        }
    }

    private fun showUpdateDialog(update: UpdateInfo) {
        val dialog = android.app.Dialog(this, android.R.style.Theme_Translucent_NoTitleBar)

        val container = FrameLayout(this).apply {
            setBackgroundColor(Color.parseColor("#80000000"))
            setPadding(dp(24), dp(24), dp(24), dp(24))
        }

        val card = cardView().apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(24), dp(28), dp(24), dp(24))

            addView(LinearLayout(this@MainActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL

                addView(TextView(this@MainActivity).apply {
                    text = "🎉"
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 32f)
                })

                addView(TextView(this@MainActivity).apply {
                    text = "发现新版本"
                    setTextColor(Color.parseColor("#0F172A"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f)
                    setTypeface(typeface, android.graphics.Typeface.BOLD)
                    setPadding(dp(12), 0, 0, 0)
                })
            })

            addView(TextView(this@MainActivity).apply {
                text = "v${update.versionName}"
                setTextColor(ContextCompat.getColor(this@MainActivity, R.color.brand_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                setPadding(0, dp(8), 0, dp(16))
            })

            if (update.forceUpdate) {
                addView(LinearLayout(this@MainActivity).apply {
                    orientation = LinearLayout.HORIZONTAL
                    gravity = Gravity.CENTER_VERTICAL
                    setPadding(0, 0, 0, dp(12))

                    addView(TextView(this@MainActivity).apply {
                        text = "⚠️"
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    })

                    addView(TextView(this@MainActivity).apply {
                        text = "此版本为强制更新，请尽快升级以获得最佳体验"
                        setTextColor(Color.parseColor("#DC2626"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                        setPadding(dp(6), 0, 0, 0)
                    })
                })
            }

            addView(View(this@MainActivity).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1))
                setBackgroundColor(Color.parseColor("#E2E8F0"))
                setPadding(0, dp(12), 0, dp(12))
            })

            addView(android.widget.ScrollView(this@MainActivity).apply {
                isVerticalScrollBarEnabled = true
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(0), 1f)
                addView(TextView(this@MainActivity).apply {
                    text = update.changelog
                    setTextColor(Color.parseColor("#475569"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 13.5f)
                    setLineSpacing(0f, 1.6f)
                })
            })

            addView(View(this@MainActivity).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1))
                setBackgroundColor(Color.parseColor("#E2E8F0"))
                setPadding(0, dp(12), 0, dp(12))
            })

            addView(TextView(this@MainActivity).apply {
                text = "立即更新"
                setTextColor(Color.WHITE)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                gravity = Gravity.CENTER
                background = android.graphics.drawable.GradientDrawable().apply {
                    setColor(ContextCompat.getColor(this@MainActivity, R.color.brand_primary))
                    cornerRadius = dp(10).toFloat()
                }
                setPadding(0, dp(14), 0, dp(14))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = dp(16)
                }
                setOnClickListener {
                    dialog.dismiss()
                    downloadUpdate(update)
                }
            })

            if (!update.forceUpdate) {
                addView(TextView(this@MainActivity).apply {
                    text = "稍后再说"
                    setTextColor(Color.parseColor("#64748B"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    gravity = Gravity.CENTER
                    setPadding(0, dp(12), 0, 0)
                    setOnClickListener { dialog.dismiss() }
                })
            }
        }

        val scrollWrapper = android.widget.ScrollView(this).apply {
            isVerticalScrollBarEnabled = false
            layoutParams = FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.WRAP_CONTENT).apply {
                gravity = Gravity.CENTER
            }
            addView(card)
        }
        container.addView(scrollWrapper)

        dialog.setContentView(container)
        dialog.setCancelable(!update.forceUpdate)

        if (update.forceUpdate) {
            dialog.setOnCancelListener { showUpdateDialog(update) }
        }

        dialog.show()
        dialog.window?.setLayout(
            resources.displayMetrics.widthPixels - dp(48),
            (resources.displayMetrics.heightPixels * 0.85f).toInt()
        )
    }

    private fun downloadUpdate(update: UpdateInfo) {
        val progressBar = ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal).apply {
            max = 100
            setPadding(dp(16), dp(16), dp(16), dp(8))
        }
        val progressText = TextView(this).apply {
            text = "准备下载..."
            setTextColor(Color.parseColor("#475569"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setPadding(dp(16), 0, dp(16), dp(16))
        }
        val layout = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            addView(progressBar)
            addView(progressText)
        }
        val dialog = AlertDialog.Builder(this)
            .setTitle("正在下载更新")
            .setView(layout)
            .setCancelable(false)
            .show()

        ApkInstaller(this).download(
            apkUrl = update.apkUrl,
            onProgress = { pct ->
                progressBar.progress = pct
                progressText.text = "下载进度：$pct%"
            },
            onComplete = { file ->
                dialog.dismiss()
                ApkInstaller(this).install(file)
            },
            onError = { msg ->
                dialog.dismiss()
                alert(msg, "下载失败")
            }
        )
    }

    companion object {
        const val TAB_HOME = 100
        const val TAB_PROJECTS = 101
        const val TAB_JD = 102
        const val TAB_TASKS = 103
        const val TAB_MORE = 104
    }
}
