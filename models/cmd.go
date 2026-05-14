package models

import (
	"bufio"
	"io"
	"os/exec"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func cmd(str string, sender *Sender) string {
	cmd := exec.Command("sh", "-c", str)
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logs.Warn("cmd.StdoutPipe: ", err)
		return err.Error()
	}
	cmd.Dir = ExecPath + "/scripts/"
	err = cmd.Start()
	if err != nil {
		logs.Warn("%v", err)
		return err.Error()
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
		msg += line
		nt := time.Now()
		if (nt.Unix() - st.Unix()) > 15 {
			sender.Reply(msg)
			st = nt
			msg = ""
		}
	}
	if msg != "" {
		sender.Reply(msg)
	}
	err = cmd.Wait()
	return msg
}




func cmd_all(str string, sender *Sender) string {
    // 修改命令字符串，确保可以执行任意目录的切换，格式为：cd <目录> && <命令>
    cmdStr := "cd " + str[:len(str)-len(str)] + " && " + str
    
    // 使用sh命令执行构造好的命令
    cmd := exec.Command("sh", "-c", cmdStr)
    stdout, err := cmd.StdoutPipe()
    stderr, err := cmd.StderrPipe()
    if err != nil {
        logs.Warn("cmd.StdoutPipe: ", err)
        return err.Error()
    }

    // 启动命令
    err = cmd.Start()
    if err != nil {
        logs.Warn("%v", err)
        return err.Error()
    }

    // 异步读取标准错误输出
    go func() {
        msg := ""
        reader := bufio.NewReader(stderr)
        for {
            // 逐行读取标准错误输出
            line, err2 := reader.ReadString('\n')
            if err2 != nil || io.EOF == err2 {
                break
            }
            msg += line
        }
        // 如果有错误输出，发送回复
        if msg != "" {
            sender.Reply(msg)
        }
    }()

    // 读取标准输出
    msg := ""
    reader := bufio.NewReader(stdout)
    for {
        // 逐行读取标准输出
        line, err2 := reader.ReadString('\n')
        if err2 != nil || io.EOF == err2 {
            break
        }
        msg += line
    }

    // 等待命令执行完成
    err = cmd.Wait()
    if err != nil {
        return err.Error()
    }

    return msg
}
