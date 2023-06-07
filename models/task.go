package models

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

type Task struct {
	ID      int
	EntryID int
	Cron    string
	Path    string
	Enable  string
	Mode    string
	Word    string
	Name    string
	Timeout int
	Args    string
	Hack    string
	Git     string
	Title   string
	Running string
	Envs    []Env `gorm:"-"`
}

func initTask() {
	for i := range Config.Tasks {
		if Config.Tasks[i].Cron != "" {
			createTask(&Config.Tasks[i])
		}
	}
}

func createTask(task *Task) {
	id, err := c.AddFunc(task.Cron, func() {
		runTask(task, &Sender{})
	})
	if err != nil {
		logs.Warn(task.Word, "任务创建失败")
	} else {
		task.ID = int(id)
		logs.Info(task.Word, "任务创建成功")
	}
}

func runTask(task *Task, sender *Sender) string {
	task.Running = True
	path := ""
	if task.Git != "" {
		path = task.Git + "/" + task.Name
	} else {
		slice := strings.Split(task.Path, "/")
		len := len(slice)
		if len == 0 {
			logs.Warn("取法识别的文件名")
			return ""
		}
		task.Name = slice[len-1]
		path = ExecPath + "/scripts/" + task.Name
		if strings.Contains(task.Path, "http") {
			f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("打开%s失败，", path, err)
				return ""
			}
			url := task.Path
			if strings.Contains(url, "raw.githubusercontent.com") {
				url = GhProxy + url
			}
			r, err := httplib.Get(url).Response()
			if err != nil {
				logs.Warn("下载%s失败，", task.Path, err)
			}
			io.Copy(f, r.Body)
			f.Close()
		} else {
			if path != task.Path && task.Name != task.Path {
				f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
				if err != nil {
					logs.Warn("打开%s失败，", path, err)
					return ""
				}
				f2, err := os.Open(task.Path)
				if err != nil {
					f.Close()
					logs.Warn("打开%s失败，", path, err)
					return ""
				}
				io.Copy(f, f2)
				f2.Close()
				f.Close()
			}
		}
	}
	lan := Config.Node
	if strings.Contains(task.Name, ".py") {
		lan = Config.Python
	}
	cmd := exec.Command(lan, task.Name)
	pins := ""
	for _, env := range GetEnvs() {
		if env.Name+".js" == task.Name && env.Value != "" {
			for _, ck := range LimitJdCookie(GetJdCookies(), env.Value) {
				pins += "&" + ck.PtPin
			}
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", "pins", pins))
	for _, env := range task.Envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logs.Warn("cmd.StdoutPipe: ", err)
		return ""
	}
	if task.Git != "" {
		cmd.Dir = task.Git
	} else {
		cmd.Dir = ExecPath + "/scripts/"
	}
	err = cmd.Start()
	if err != nil {
		logs.Warn("%v", err)
		return ""
	}
	go func() {
		msg := ""
		reader := bufio.NewReader(stderr)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			msg += line

		}
		if msg != "" {
			sender.Reply(msg)
		}
	}()
	msg := ""
	reader := bufio.NewReader(stdout)
	st := time.Now()
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		if task.Name == "jd_get_share_code.js" {
			rt := findShareCode(line)
			if rt != "" {
				sender.Reply(rt)
			}
		}
		msg += line
		nt := time.Now()
		if (nt.Unix() - st.Unix()) > 1 {
			sender.Reply(msg)
			st = nt
			msg = ""
		}
	}
	if msg != "" {
		logs.Info("消息测试")
		if (task.Name == "jd_qmckd_branchHelp.js" || task.Name == "jd_qmckd_taskHelp.js") && strings.Contains(msg, "本次共运行") {
			ss := regexp.MustCompile(`(\d+)`).FindStringSubmatch(msg)
			logs.Info(ss)
			atoi, err := strconv.Atoi(ss[1])
			if err != nil {
				panic(err)
			}
			rsp := DeleteCk(atoi)
			sender.Reply(rsp)
		}
		sender.Reply(msg)
	}
	task.Running = False
	return msg
}

func findShareCode(msg string) string {
	o := false
	for _, v := range regexp.MustCompile(`京东账号\d*（(.*)）(.*)】(\S*)`).FindAllStringSubmatch(msg, -1) {
		if !strings.Contains(v[3], "种子") && !strings.Contains(v[3], "undefined") {
			pt_pin := url.QueryEscape(v[1])
			for key, ss := range map[string][]string{
				"Fruit":        {"京东农场", "东东农场"},
				"Pet":          {"京东萌宠"},
				"Bean":         {"种豆得豆"},
				"JdFactory":    {"东东工厂"},
				"DreamFactory": {"京喜工厂"},
				"Jxnc":         {"京喜农场"},
				"Jdzz":         {"京东赚赚"},
				"Joy":          {"crazyJoy"},
				"Sgmh":         {"闪购盲盒"},
				"Cfd":          {"财富岛"},
				"Cash":         {"签到领现金"},
			} {
				for _, s := range ss {
					if strings.Contains(v[2], s) && v[3] != "" {
						if ck, err := GetJdCookie(pt_pin); err == nil {
							ck.Update(key, v[3])
						}
						if !o {
							o = true
						}
					}
				}
			}
		}
	}
	if o {
		return "导入互助码成功"
	} else {
		return ""
	}
}

func DeleteCk(N int) string {
	N -= 2
	if N > 0 {
		// 打开原始文件和临时文件
		inputFile, err := os.Open(ExecPath + "/scripts/ck.txt")
		if err != nil {
			panic(err)
		}
		defer inputFile.Close()

		tmpFile, err := os.CreateTemp(ExecPath+"/scripts", "temp")
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// 创建一个 Scanner 对象来逐行读取原始文件
		scanner := bufio.NewScanner(inputFile)

		// 跳过前 N 行
		for i := 0; i < N; i++ {
			if !scanner.Scan() {
				// 如果文件行数不足 N 行，则直接退出
				return " 如果文件行数不足 N 行，直接退出"
			}
		}

		// 将剩余的行写入临时文件
		for scanner.Scan() {
			fmt.Fprintln(tmpFile, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}

		// 关闭原始文件和临时文件
		inputFile.Close()
		tmpFile.Close()

		// 删除原有的文件
		err = os.Remove(ExecPath + "/scripts/ck.txt")
		if err != nil {
			panic(err)
		}

		// 重命名临时文件为原始文件
		err = os.Rename(tmpFile.Name(), ExecPath+"/scripts/ck.txt")
		if err != nil {
			panic(err)
		}
		//return fmt.Sprintf("成功删除%d行", N)
		return "删除成功"
	}
	return "低于0不删除"

}
