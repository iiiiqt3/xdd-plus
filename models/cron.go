package models

import (
	"fmt"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/robfig/cron/v3"
	"math/rand"
	"strconv"
	"time"
)

var c *cron.Cron

func initCron() {
	c = cron.New()
	if Config.DailyAssetPushCron != "" {
		_, err := c.AddFunc(Config.DailyAssetPushCron, DailyAssetsPush)
		if err != nil {
			logs.Warn("资产推送任务失败：%v", err)
		} else {
			logs.Info("资产推送任务就绪")
		}

		//c.AddFunc("3 */1 * * *", initVersion)
		//c.AddFunc("40 */1 * * *", GitPullAll)
	}
	if Config.DailyCompletePush != "" {
		c.AddFunc(Config.DailyCompletePush, CompletePush)
	}

	spec := GetEnv("cron")
	if spec == "" {
		spec = strconv.Itoa(rand.Intn(59)) + " 10/12 * * ?" //cron表达式，每秒一次
	}
	c.AddFunc(spec, func() {
		fmt.Println("开始wskey转换")
		updateCookie()
		UpdateRwskey()
	})

	c.AddFunc(strconv.Itoa(rand.Intn(59))+" "+strconv.Itoa(rand.Intn(24))+" * * ?", getAuthFlag)
	c.AddFunc(strconv.Itoa(rand.Intn(59))+" 10 5/7 * ?", GetAuthKey)
	//logs.Info("0 " + strconv.Itoa(rand.Intn(59)) + " 0/" + strconv.Itoa(Config.Later) + " * * ?" + "调试推送时间")
	c.AddFunc("0 8-20/5 * * ?", initCookie)
	c.AddFunc("0 10 * * 3,5", AutoBak)
	c.AddFunc("0 8-20/1 * * ?", GetNewVersion)

	c.Start()
}

func initHELP() {
	orderQueue = &OrderQueue{}
	for {
		order := orderQueue.ProcessOrder()
		if order == nil {
			logs.Info("ProcessOrder")
			break
		}

		fmt.Printf("Processing order %d: %s\n", order.id, order.name)
		time.Sleep(time.Second * time.Duration(20))
	}
}
