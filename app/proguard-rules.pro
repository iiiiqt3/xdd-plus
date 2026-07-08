-keep class com.feiniao.androidjd.data.model.** { *; }
-dontwarn org.codehaus.mojo.animal_sniffer.IgnoreJRERequirement

-dontwarn cn.jpush.**
-keep class cn.jpush.** { *; }
-keep class * extends cn.jpush.android.service.JPushMessageReceiver { *; }
-dontwarn cn.jiguang.**
-keep class cn.jiguang.** { *; }
