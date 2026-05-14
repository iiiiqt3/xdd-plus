package models

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
)

func initHandle() {
	//获取路径
	Save = make(chan *JdCookie)
	go func() {
		init := true
		for {
			get := <-Save
			if get.Pool == "s" {
				//initCookie()
				continue
			}
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)
			})

			logs.Info(fmt.Sprintf("总共%d个号", len(cks)))
			var tmp []JdCookie
			for _, ck := range cks {
				if ck.Priority >= 0 {
					tmp = append(tmp, ck)
				}
			}
			cks = tmp
			cookies := "{"
			var hh []string
			for i, ck := range cks {
				hh = append(hh,
					fmt.Sprintf("CookieJD%d:'pt_key=%s;pt_pin=%s;'", i+1, ck.PtKey, ck.PtPin),
				)
			}
			cookies += strings.Join(hh, ",")
			cookies += "}"
			f, err := os.OpenFile(ExecPath+"/scripts/jdCookie.js", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.js失败，", err)
			}
			f1, err := os.OpenFile(ExecPath+"/scripts/6dylan6_jdpro/jdCookie.js", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.js失败，", err)
			}
              f2, err := os.OpenFile(ExecPath+"/scripts/feverrun_my_scripts/jdCookie.js", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.js失败，", err)
			}
			
			f2.WriteString(fmt.Sprintf(`
var cookies = %s
var pins = process.env.pins
if(pins){
    pins = pins.split("&")
    for (var key in cookies) {
        c = false
        for (var pin of pins) {
            if (pin && cookies[key].indexOf(pin) != -1) {
                c = true
                break
            }
        }
        if (!c) {
            delete cookies[key]
        }
    }
}
module.exports = cookies`, cookies))

			f1.WriteString(fmt.Sprintf(`
var cookies = %s
var pins = process.env.pins
if(pins){
    pins = pins.split("&")
    for (var key in cookies) {
        c = false
        for (var pin of pins) {
            if (pin && cookies[key].indexOf(pin) != -1) {
                c = true
                break
            }
        }
        if (!c) {
            delete cookies[key]
        }
    }
}
module.exports = cookies`, cookies))


			

			f.WriteString(fmt.Sprintf(`
var cookies = %s
var pins = process.env.pins
if(pins){
	pins = pins.split("&")
	for (var key in cookies) {
	    c = false
	    for (var pin of pins) {
		   if (pin && cookies[key].indexOf(pin) != -1) {
			  c = true
			  break
		   }
	    }
	    if (!c) {
		   delete cookies[key]
	    }
	}
}
module.exports = cookies`, cookies))
			f.Close()
		     f1.Close() 
		     f2.Close() // 完成操作后记得关闭文件
			//WriteHelpJS(cks)
			go CopyConfigAll()
			// tmp = []JdCookie{}
			// for _, ck := range cks {
			// 	if ck.Hack != True {
			// 		tmp = append(tmp, ck)
			// 	}
			// }
			// cks = tmp
			if Config.Mode == Parallel {
				for i := range Config.Containers {
					(&Config.Containers[i]).read()
				}
				for i := range Config.Containers {
					(&Config.Containers[i]).write(cks)
				}
			} else if Config.Mode == Vip {
              
				if Config.VIP {
					balanceIndices := []int{}  // 存储所有 Mode == Balance 的容器索引
					residentCkpin := make(map[string]bool) //存储车头ck pin
					cl := 0
					sl := 0 // 新增：统计 Special 模式可用容器的数量
					logs.Info("进入VIP模式")
					// 遍历容器，读取容器配置并清空每个容器的 cookies
					for i := range Config.Containers {
						(&Config.Containers[i]).read()
						Config.Containers[i].cks = []JdCookie{}
						// 如果容器可用且模式为 Balance，则统计 Balance 容器的数量
						if Config.Containers[i].Available {
							if Config.Containers[i].Mode == Balance {
								cl++
								balanceIndices = append(balanceIndices, i)
								ctpin := Config.Containers[i].Resident
								if ctpin != ""{
									for k := range cks {
										ck := cks[k]
										if strings.Contains(ctpin, ck.PtPin) {
											if ck.Hack == True {
												continue
											}
											Config.Containers[i].cks = append(Config.Containers[i].cks, ck)
											residentCkpin[ck.PtPin] = true
										}	
									}
								}
							} else if Config.Containers[i].Mode == Special {
								sl++ // 统计 Special 模式可用容器的数量
							}
						}
					}
					if cl != 0 {
						for i := range cks {
							ck := cks[i]
							// 判断 ck.Hack 是否为 true，若为 true，则跳过该 cookie，不进行负载均衡
							if ck.Hack == True {
								// 如果 Appoint 为 true，则进入 Special 容器
								if ck.Appoint == True {
									// 如果没有可用的 Special 容器，打印日志信息
									if sl == 0 {
										logs.Warn("没有可用的 Special 容器来分配该 cookie")
										continue
									}
									assigned := false
									for j := range Config.Containers {

										if Config.Containers[j].Available && Config.Containers[j].Mode == Special {
										//    logs.Info(fmt.Sprintf("当前洗白容器：%d", Config.Containers[j].Address))
											Config.Containers[j].cks = append(Config.Containers[j].cks, ck)
											assigned = true
											break
										}
									}
									if !assigned {
										logs.Warn("没有找到适合的 Special 容器来分配该 cookie")
									}
								}
								// 如果 Hack 为 true，无论 Appoint 是否为 true，都跳过该 cookie，不进行负载均衡
								continue
							}
							// 如果 Hack 为 false，正常进行 Balance 容器分配
							//j := i % cl
							//logs.Info(fmt.Sprintf("当前平行容器：%d", Config.Containers[j].Address))
							//Config.Containers[j].cks = append(Config.Containers[j].cks, ck)
							j := i % cl
							targetIdx := balanceIndices[j]  // 通过预存索引获取真实容器位置
							if !residentCkpin[ck.PtPin] {
								Config.Containers[targetIdx].cks = append(Config.Containers[targetIdx].cks, cks[i])
							}	
						}
					}
					// 将分配好的 cookies 写入容器
					for i := range Config.Containers {
						if Config.Containers[i].Available {
						//	logs.Info(fmt.Sprintf("写入容器：%d", Config.Containers[i].Address))
							if Config.Containers[i].Mode == Balance {
								// 写入 Balance 模式容器的 cookies
								(&Config.Containers[i]).write(Config.Containers[i].cks)
							} else if Config.Containers[i].Mode == Special {
								// 写入 Special 模式容器的 cookies
								(&Config.Containers[i]).write(Config.Containers[i].cks)
							} else {
								// Parallel 模式下，不考虑 Hack 和 Appoint 的值，直接写入
								(&Config.Containers[i]).write(cks)
							}
						}
					}
				}
         
                

			} else {
				var resident []JdCookie

				//不影响原本的设置车头逻辑,在容器内单独配置车头，并且可以覆盖全局的车头
				var containerResident []string
				for _, container := range Config.Containers {
					if container.Resident != "" {
						containerResident = append(containerResident, container.Resident)
					}
				}

				//为了过滤设置头的ck并且不影响原来逻辑
				if len(containerResident) > 0 {
					//如果配置了单独的车头，则全局不生效
					Config.Resident = strings.Join(containerResident, "&")
				}

				var residentCkMap = make(map[string]JdCookie)

				if Config.Resident != "" {
					tmp := cks
					cks = []JdCookie{}
					for _, ck := range tmp {
						if strings.Contains(Config.Resident, ck.PtPin) {
							resident = append(resident, ck)
							residentCkMap[ck.PtPin] = ck
						} else {
							cks = append(cks, ck)
						}
					}
				}
				type balance struct {
					Container Container
					Weigth    float64
					Ready     []JdCookie
					Resident  []JdCookie
					Should    int
				}
				var availables []Container
				var parallels []Container
				var bs []balance
				for i := range Config.Containers {
				(&Config.Containers[i]).read()
					if Config.Containers[i].Available {
						if Config.Containers[i].Mode == Parallel {
							parallels = append(parallels, Config.Containers[i])
						} else {
							availables = append(availables, Config.Containers[i])
							bs = append(bs, balance{
								Container: Config.Containers[i],
								Weigth:    float64(Config.Containers[i].Weigth),
							})
						}
					}
				}
				bat := cks
				//是否配置了优先级
				if Config.Priority == 0 {
					for {
						left := []JdCookie{}
						l := len(cks)
						total := 0.0
						for i := range bs {
							total += float64(bs[i].Weigth)
						}
						for i := range bs {
							if bs[i].Weigth == 0 {
								bs[i].Should = 0
							} else {
								bs[i].Should = int(math.Ceil(bs[i].Weigth / total * float64(l)))
							}

						}
						a := 0
						for i := range bs {
							j := bs[i].Should
							if j == 0 {
								continue
							}
							s := 0
							if bs[i].Container.Limit > 0 && j > bs[i].Container.Limit {
								s = a + bs[i].Container.Limit
								left = append(left, cks[s:a+j]...)
								bs[i].Weigth = 0
							} else {
								s = a + j
							}
							if s > l {
								s = l
							}
							bs[i].Ready = append(bs[i].Ready, cks[a:s]...)
							a += j
							if a >= l-1 {
								break
							}
						}
						if len(left) != 0 {
							cks = left
							continue
						}
						break
					}
				} else {
					var ups []JdCookie
					var downs []JdCookie
					for _, ck := range cks {
						if ck.Priority >= Config.Priority {
							ups = append(ups, ck)
						} else {
							downs = append(downs, ck)
						}
					}
					zhuCks := 0
					ciCks := 0
					for i := 0; i < len(bs); i++ {
						//处理车头
						//获取容器配置中的车头
						containerResident := bs[i].Container.Resident
						var bsResident []JdCookie
						if containerResident != "" {
							containerResidents := strings.Split(containerResident, "&")
							for _, cr := range containerResidents {
								bsResident = append(bsResident, residentCkMap[cr])
							}
							bs[i].Resident = bsResident
						}

						//最后一个，ck全部放进去
						if i == len(bs)-1 {
							bs[i].Ready = append(bs[i].Ready, ups[zhuCks:]...)
							bs[i].Ready = append(bs[i].Ready, downs[ciCks:]...)
						} else {
							//必须配置了main的数量，并且main的数量要小于总数
							s := 0
							s = int(math.Ceil(float64(len(ups) / len(bs))))
							if s > len(ups) {
								s = len(ups)
							}
							bs[i].Ready = append(bs[i].Ready, ups[zhuCks:s+zhuCks]...)
							zhuCks += s

							s = 0
							s = int(math.Ceil(float64(len(downs) / len(bs))))
							if s > len(downs) {
								s = len(downs)
							}

							bs[i].Ready = append(bs[i].Ready, downs[ciCks:s+ciCks]...)
							ciCks += s
						}
					}
				}
				for i := range bs {
					if len(bs[i].Resident) != 0 {
						bs[i].Container.write(append(bs[i].Resident, bs[i].Ready...))
					} else {
						bs[i].Container.write(append(resident, bs[i].Ready...))
					}
				}
				for i := range parallels {
					parallels[i].write(append(resident, bat...))
				}
			}
			if init {
				go func() {
					for {
						Save <- &JdCookie{
							Pool: "s",
						}
						time.Sleep(time.Minute * 30)
						// time.Sleep(time.Second * 1)
					}
				}()
				init = false
			}
		}
	}()
}

func randSlice(slice interface{}) { //切片乱序
	rv := reflect.ValueOf(slice)
	if rv.Type().Kind() != reflect.Slice {
		return
	}

	length := rv.Len()
	if length < 2 {
		return
	}

	swap := reflect.Swapper(slice)
	rand.Seed(time.Now().Unix())
	for i := length - 1; i >= 0; i-- {
		j := rand.Intn(length)
		swap(i, j)
	}
	return
}
