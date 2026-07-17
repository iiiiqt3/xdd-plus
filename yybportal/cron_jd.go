package yybportal

import (
	"github.com/robfig/cron/v3"
)

var jdCron *cron.Cron

func startJdCron() {
	if jdCron != nil {
		return
	}
	jdCron = cron.New()
	jdCron.AddFunc("0 */3 * * ?", RefreshYybCKAuto)
	jdCron.AddFunc("* * * * *", runYybLivenessScheduleTick)
	jdCron.Start()
}
