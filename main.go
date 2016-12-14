package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/larspensjo/config"
	"github.com/qiniu/iconv"
	_ "github.com/wendal/go-oci8"
)

var tape = readLogin()
var sysdate = time.Now().Format("2006-01-02")
var filename = sysdate + ".log"

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU()) //使用满核
	date1, date2, ip1, ip2 := getprocessing_date()
	checkdate(date1, sysdate, ip1)
	checkdate(date2, sysdate, ip2)
	mail()
}
func readLogin() map[string]string {
	var (
		configFile = flag.String("configfile", "config.ini", "General configuration file")
	)
	var TOPIC1 = make(map[string]string)
	flag.Parse()
	cfg, err := config.ReadDefault(*configFile)
	if err != nil {
		log.Fatalf("Fail to find", *configFile, err)
	}
	if cfg.HasSection("loginOracle") {
		section, err := cfg.SectionOptions("loginOracle")
		if err == nil {
			for _, v := range section {
				options, err := cfg.String("loginOracle", v)
				if err == nil {
					cd, err := iconv.Open("utf-8", "gbk")
					erro(err)
					defer cd.Close()
					tmp := cd.ConvString(options)
					TOPIC1[v] = tmp
				}
			}
		}
	}
	return TOPIC1
}
func getprocessing_date() (string, string, string, string) {
	processing_date := loginOracle(tape["login"], tape["sql"])
	processing_date2 := loginOracle(tape["login2"], tape["sql"])
	return processing_date, processing_date2, tape["ip"], tape["ip2"]
}
func loginOracle(logininfo string, sqlinfo string) string {
	var Data_value string
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	os.Setenv("NLS_LANG", "")
	db, err := sql.Open("oci8", logininfo)
	erro(err)
	defer db.Close()
	rows, err := db.Query(sqlinfo)
	columns, err := rows.Columns()
	erro(err)
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		erro(err)
		var value string
		for _, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
		}
		Data_value = value
	}
	return Data_value
}
func checkdate(date string, sys string, ip string) {
	dstfile, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE, 0777)
	erro(err)
	defer dstfile.Close()
	logtest := log.New(dstfile, "\r\n", log.LstdFlags)
	if date == sys {
		logtest.Println("数据库ip:" + ip + "  processing_date与系统时间一致,processing_date：" + date)
	} else {
		logtest.Println("数据库ip:" + ip + "  processing_date与系统时间不一致,processing_date：" + date)
	}
}
func mail() {
	dstfile, err := os.Open(filename)
	erro(err)
	defer dstfile.Close()
	comment, _ := ioutil.ReadAll(dstfile)
	body := string(comment)
	fmt.Println("正在发送")
	err1 := SendMail(tape["user"], tape["password"], tape["host"], tape["to"], tape["subject"], body, "text")
	erro(err1)
}
func SendMail(user, password, host, to, body, subject, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + user + "<" + user + ">\r\n" + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	return err
}
func erro(err error) {
	if err != nil {
		fmt.Println("出错了", err)
	}
}
