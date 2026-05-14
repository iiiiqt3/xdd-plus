package models

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/golang/freetype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	dpi      = flag.Float64("dpi", 72, "screen resolution in Dots Per Inch")
	fontfile = flag.String("fontfile", "simsunb.ttf", "filename of the ttf font")
	hinting  = flag.String("hinting", "full", "none | full")
	size     = flag.Float64("size", 20, "font size in points")
	spacing  = flag.Float64("spacing", 1.2, "line spacing (e.g. 2 means double spaced)")
	wonb     = flag.Bool("whiteonblack", false, "white text on a black background")
)

func strtoimg(str string) []byte {
	log.Info("开始转换图片")
	text := strings.Split(str, "\n")
	fontBytes, err := ioutil.ReadFile(*fontfile)
	if err != nil {
		log.Println(err)
		return nil
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return nil
	}

	// Initialize the context.
	fg, bg := image.Black, image.White
	if *wonb {
		fg, bg = image.White, image.Black
	}
	rgba := image.NewRGBA(image.Rect(0, 0, 420, 500))
	draw.Draw(rgba, rgba.Bounds(), bg, image.Point{}, draw.Src)
	c := freetype.NewContext()
	c.SetDPI(*dpi)
	c.SetFont(f)
	c.SetFontSize(*size)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	switch *hinting {
	default:
		c.SetHinting(font.HintingNone)
	case "full":
		c.SetHinting(font.HintingFull)
	}

	// Draw the text.
	pt := freetype.Pt(10, 10+int(c.PointToFixed(*size)>>6))
	for _, s := range text {
		_, err = c.DrawString(s, pt)
		if err != nil {
			log.Println(err)
			return nil
		}
		pt.Y += c.PointToFixed(*size * *spacing)
	}
	//_, err = c.DrawString(text, pt)

	fileName := fmt.Sprintf("%daaa.png", time.Now().Unix())
	outFile, err := os.Create(fileName)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	file, _ := os.ReadFile(fileName)
	os.Remove(fileName)
	return file
}

func uploadImg(filename string) string {
	if sysConfig.ImageToken == "" {
		logs.Info("图床Token为空")
		if sysConfig.ImageUserName == "" || sysConfig.ImagePassword == "" {
			logs.Info("图床账号密码为空")
			return "图床账号密码为空"
		} else {
			GetImageToken()
		}
	}

	get := httplib.Post("http://180.152.5.230:5703/api/v1/upload")
	get.Header("Authorization", "Bearer "+sysConfig.ImageToken)
	get.Header("Content-Type", "multipart/form-data")
	get.Param("strategy_id", "1")
	get.PostFile("file", filename)
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
	s, _ := jsonparser.GetString(bytes, "data", "links", "url")
	logs.Info(s)
	return s

}

func GetImageToken() {
	get := httplib.Post("http://180.152.5.230:5703/api/v1/tokens")
	get.Header("Content-Type", "multipart/form-data")
	get.Param("email", sysConfig.ImageUserName)
	get.Param("password", sysConfig.ImagePassword)
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
	val, _ := jsonparser.GetBoolean(bytes, "status")
	if val {
		token, _ := jsonparser.GetString(bytes, "data", "token")
		sysConfig.ImageToken = token
		SaveSysConfig(sysConfig)
		logs.Info("图床登录成功，已记录Token")
	} else {
		logs.Warn("图床登录失败，请检查用户名密码")
	}

}
