package models

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Autologin() {
	initChrome()
	initWebDisplay()
	login()

}

func initChrome() {
	var chromePath string
	if runtime.GOOS == "windows" {
		chromePath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "pyppeteer", "pyppeteer", "local-chromium", "588429", "chrome-win32", "chrome.exe")
	} else if runtime.GOOS == "linux" {
		chromePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "pyppeteer", "local-chromium", "1181205", "chrome-linux", "chrome")
	} else if runtime.GOOS == "darwin" {
		fmt.Println("mac")
		return
	} else {
		fmt.Println("unknown")
		return
	}

	if _, err := os.Stat(chromePath); os.IsNotExist(err) {
		fmt.Println("Chrome is not installed")

		//todo Add your code to download and install Chrome
	} else {
		fmt.Println("Chrome is installed")
	}
}

var WebDisplay = true
var configfile = "config.ini"

func initWebDisplay() {
	file, err := os.Open(configfile)
	if err != nil {
		fmt.Println("读取配置文件时出错")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Displaylogin=1") {
			WebDisplay = false
			fmt.Println("当前模式：显示web登录图形化界面")
			break
		}
	}

	if WebDisplay {
		fmt.Println("当前配置不显示web登录图形化界面，若要取消静默登陆，在配置文件中设置参数Displaylogin=1")
	}
}

func humanType(ctx context.Context, sel, input string) error {
	for _, c := range input {
		err := chromedp.SendKeys(sel, string(c), chromedp.ByID).Do(ctx)
		if err != nil {
			return err
		}
		// 模拟人的行为，随机延迟
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}
	return nil
}

func login() {
	// Replace with your own usernum, passwd, and notes
	usernum := "18065858679"
	passwd := "764763903a"
	notes := "yourNotes"

	log.Printf("正在登录 %s %s 的账号", notes, usernum)

	// 指定浏览器路径
	//browserPath := "C:\\Users\\zjy\\AppData\\Local\\pyppeteer\\pyppeteer\\local-chromium\\1181205\\chrome-win\\chrome.exe"

	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // 设置为非无头模式，即可视化浏览器界面
		//chromedp.ExecPath(browserPath),   // 指定浏览器路径
		// Add more flags as needed
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	var cookies []*network.Cookie
	var searchWrapper bool
	var subtitle bool
	//var sval string

	var nodes []*cdp.Node

	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(360, 640), // 设置视窗大小
		chromedp.Navigate(`https://plogin.m.jd.com/login/login?appid=300&returnurl=https%3A%2F%2Fm.jd.com%2F&source=wq_passport`),
		chromedp.WaitVisible(`.J_ping.planBLogin`, chromedp.ByQuery),
		chromedp.Click(`.J_ping.planBLogin`, chromedp.ByQuery),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return humanType(ctx, `#username`, usernum)
		}),
		chromedp.Sleep(time.Duration(rand.Intn(3))*time.Second),
		chromedp.WaitVisible(`#pwd`, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return humanType(ctx, `#pwd`, passwd)
		}),
		chromedp.Sleep(time.Duration(rand.Intn(3))*time.Second),
		chromedp.Click(`.policy_tip-checkbox`, chromedp.ByQuery),
		chromedp.Sleep(time.Duration(rand.Intn(2))*time.Second),
		chromedp.Click(`.btn.J_ping.btn-active`, chromedp.ByQuery),
		chromedp.Sleep(time.Duration(rand.Intn(5))*time.Second),
		chromedp.Nodes("#small_img", &nodes, chromedp.ByQuery),
	)

	var cpcimg string
	var smallimg string
	if len(nodes) > 0 {
		log.Println("滑块存在，需要处理滑块验证")
		var boxModel *dom.BoxModel
		chromedp.Run(ctx,

			chromedp.AttributeValue(`#cpc_img`, "src", &cpcimg, nil),
			// 下载滑块背景图并处理
			chromedp.AttributeValue(`#small_img`, "src", &smallimg, nil),

			chromedp.Nodes(".move-img", &nodes, chromedp.ByQuery),

			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				boxModel, err = dom.GetBoxModel().WithNodeID(nodes[0].NodeID).Do(ctx)
				if err != nil {
					return err
				}
				return nil
			}),
		)
		if cpcimg != "" && smallimg != "" {
			// 通过GetXY获得坐标并模拟移动
			distance := int(getDistance(cpcimg, smallimg))

			// 计算滑块的中心点
			x := int64(boxModel.Margin[0] + 25)
			y := int64(boxModel.Margin[1] + 25)
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)

			chromedp.Run(ctx,
				// 模拟鼠标按下滑块
				chromedp.ActionFunc(func(ctx context.Context) error {
					return input.EmulateTouchFromMouseEvent(input.MousePressed, x, y, "left").Do(ctx)
				}),

				// 模拟滑块滑动
				chromedp.ActionFunc(func(ctx context.Context) error {

					// 增加人的不确定性

					intn := rand.Intn(20) + 15
					for i := 0; i < distance+intn; i++ {
						x++
						err := input.EmulateTouchFromMouseEvent(input.MouseMoved, x, y, "left").Do(ctx)
						if err != nil {
							return err
						}
						// 可以在这里添加一些随机的延迟来模拟人的行为
						time.Sleep(time.Duration(rand.Intn(15)) * time.Millisecond)

					}

					for i := 0; i < intn; i++ {
						x--
						err := input.EmulateTouchFromMouseEvent(input.MouseMoved, x, y, "left").Do(ctx)
						if err != nil {
							return err
						}
						// 可以在这里添加一些随机的延迟来模拟人的行为
						time.Sleep(time.Duration(rand.Intn(15)) * time.Millisecond)

					}

					return nil
				}),

				// 模拟鼠标放下滑块
				chromedp.ActionFunc(func(ctx context.Context) error {
					return input.EmulateTouchFromMouseEvent(input.MouseReleased, x, y, "left").Do(ctx)
				}),

				chromedp.ActionFunc(func(ctx context.Context) error {
					var err error
					cookies, err = storage.GetCookies().Do(ctx)
					if err != nil {
						return err
					}
					return nil
				}),
			)

		}
	} else {
		log.Println("滑块不存在，无需处理滑块验证")
	}

	var ptkey string
	var ptpin string
	for _, cookie := range cookies {
		if cookie.Name == "pt_key" {
			ptkey = cookie.Value
		}
		if cookie.Name == "pt_pin" {
			ptpin = cookie.Value
		}
	}
	if ptpin != "" && ptkey != "" {
		logs.Info("登陆成功")
		logs.Info(fmt.Sprintf("pt_key: %s", ptkey))
	} else {
		logs.Error("登陆失败")

	}

	if err != nil {
		log.Fatal(err)
	}

	if searchWrapper {
		// Here you would call your SubmitCK function
		// Since this function involves user interaction and is not directly related to chromedp,
		// it is not included in this example.
		log.Printf("Submitting CK for %s", notes)
		// Close the browser
		cancel()
	}

	if subtitle {
		fmt.Println("需要进行短信验证")
		// Here you would call your push_message and get_user_choice functions
		// and handle the user's choice accordingly.
		// Since these functions involve user interaction and are not directly related to chromedp,
		// they are not included in this example.
	}

}

func downloadImage(url string) ([]byte, error) {
	// Data URIs look like "data:image/png;base64,iVBORw0KGg...."
	// We need to split the string on the comma to separate the metadata from the actual data
	parts := strings.SplitN(url, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URI")
	}

	// The actual data is base64-encoded, so we need to decode it
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getDistance(backgroundImageUrl string, sliderImageUrl string) float64 {

	backgroundImg, err := downloadImage(backgroundImageUrl)
	if err != nil {
		log.Fatal(err)
	}

	sliderImg, err := downloadImage(sliderImageUrl)
	if err != nil {
		log.Fatal(err)
	}

	img, _ := gocv.IMDecode(backgroundImg, gocv.IMReadGrayScale)
	template, _ := gocv.IMDecode(sliderImg, gocv.IMReadGrayScale)

	gocv.GaussianBlur(img, &img, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
	gocv.GaussianBlur(template, &template, image.Pt(5, 5), 0, 0, gocv.BorderDefault)

	bgEdge := gocv.NewMat()
	cutEdge := gocv.NewMat()

	gocv.Canny(img, &bgEdge, 100, 200)
	gocv.Canny(template, &cutEdge, 100, 200)

	bgEdge.ConvertTo(&img, gocv.MatTypeCV8UC3)
	cutEdge.ConvertTo(&template, gocv.MatTypeCV8UC3)

	res := gocv.NewMat()
	gocv.MatchTemplate(img, template, &res, gocv.TmCcorrNormed, gocv.NewMat())

	_, _, _, maxLoc := gocv.MinMaxLoc(res)
	distance := maxLoc.X + 10

	return float64(distance)
}
