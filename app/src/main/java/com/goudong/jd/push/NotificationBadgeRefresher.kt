package com.goudong.jd.push

import java.util.concurrent.CopyOnWriteArrayList

object NotificationBadgeRefresher {
    private val listeners = CopyOnWriteArrayList<() -> Unit>()

    fun register(listener: () -> Unit) {
        listeners.addIfAbsent(listener)
    }

    fun unregister(listener: () -> Unit) {
        listeners.remove(listener)
    }

    fun refresh() {
        listeners.forEach { listener ->
            runCatching { listener() }
        }
    }
}
