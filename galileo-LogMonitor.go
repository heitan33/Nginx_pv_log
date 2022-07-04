package main

import (
	"fmt"
//	"reflect"
	"sync"
	"strings"
	"net/http"
    "encoding/json"
	"os"
	"bufio"
	"io"
	"time"
    "gopkg.in/yaml.v2"
	"io/ioutil"
	"github.com/hpcloud/tail"
)


type conf struct {
	VisitUrl  	string 		`yaml:"visitUrl"`
}


func (c *conf) getConf() *conf {
	yamlFile ,err := ioutil.ReadFile("visitVolume.yaml")
	if err != nil {
		fmt.Println("yamlFile.Get err", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println("Unmarshal: ", err.Error())
	}
	return c
}


func Post(data ,visitUrl string) {
	jsoninfo := strings.NewReader(data)
	client := &http.Client{}
	req, err := http.NewRequest("POST", visitUrl, jsoninfo)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", "xxx")
	resp, err := client.Do(req)
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			return
	}
		fmt.Println("Process panic done Post")
	}()
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(body))
	fmt.Println(resp.StatusCode)
}


var properties = make(map[string]string)
func init() {
	srcFile, err := os.OpenFile("./visitVolume.properties", os.O_RDONLY, 0666)
	defer srcFile.Close()
	if err != nil {
		fmt.Println("The file not exits.")
	} else {
		srcReader := bufio.NewReader(srcFile)
		for {
			str, err := srcReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			if len(strings.TrimSpace(str)) == 0 || str == "\n" {
				continue
			} else {
				fmt.Println(str)
				num++
				properties[strings.Replace(strings.Split(str, ":")[0], " ", "", -1)] = strings.Replace(strings.Split(str, ":")[1], " ", "", -1)
			}
		}
	}
}


type postData struct {
	ConfigId 		string 		`json:"ConfigId"`
	Volume 			int 		`json:"volume"`
}

var wg sync.WaitGroup
var num int = 0
func main() {
	var config conf
	urlConfig := config.getConf()
	visitUrl := urlConfig.VisitUrl
	wg.Add(num)
	for machineId ,logAbsPath := range properties {
		machineId = strings.TrimSpace(strings.Replace(machineId, "\n", "" ,-1))
		logAbsPath = strings.TrimSpace(strings.Replace(logAbsPath, "\n", "" ,-1))
		go tailLog(logAbsPath ,machineId ,visitUrl)
	}
	wg.Wait()
}

func tailLog (logAbsPath ,machineId ,visitUrl string) {
	var startTime ,endTime int64
	var count int
	var data postData

	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	tails, err := tail.TailFile(logAbsPath, config)
	if err != nil {
		fmt.Println("tail file failed, err:", err)
		return
	}
	var (
		line *tail.Line
		ok   bool
	)
	startTime = time.Now().Unix()
	for {
		line, ok = <-tails.Lines 	//遍历chan，读取日志内容
		count++
		if !ok {
			fmt.Printf("tail file close reopen, filename:%s\n", tails.Filename)
			continue
		}
		fmt.Println("line:", line.Text)
		endTime = time.Now().Unix()
		if endTime - startTime >= 60 {
			data = postData{ConfigId: machineId, Volume: count}
			dataJson, err := json.Marshal(data)
			if err != nil {
				fmt.Println("data json trans err", err.Error())
			}
			dataJsonStr := string(dataJson)
			fmt.Println(dataJsonStr)
			Post(dataJsonStr ,visitUrl)
			count = 0
			startTime = endTime
		}
	}
	wg.Done()
}
