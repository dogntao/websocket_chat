package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type message struct {
	fromUser    string
	toUser      string
	contentType int
	content     string
	createTime  string
}

// 消息队列
var messageQueue = make(chan *message, 20)

// 用户和websocket连接对应关系(用于给谁发消息)
var userConMap = make(map[string]*websocket.Conn)

// websocket连接和用户对应关系(用于判断用户名)
var conUserMap = make(map[*websocket.Conn]string)

var homeTpl = template.Must(template.ParseFiles("home.html"))

// 首页
func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTpl.Execute(w, r.Host)
}

// websocket连接
func ws(w http.ResponseWriter, r *http.Request) {
	con, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	conUserMap[con] = ""
	if err != nil {
	}
	for {
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		mt, mg, _ := con.ReadMessage()
		if strings.HasPrefix(string(mg), "username=") {
			// 登录
			userLoinArr := strings.Split(string(mg), "=")
			if userLoinArr[1] != "" {
				// 用户和websocket连接对应关系
				userConMap[userLoinArr[1]] = con
				// websocket连接和用户对应关系
				conUserMap[con] = userLoinArr[1]
				// 登录提示
				message := &message{"system", "all", mt, "welcome " + userLoinArr[1], currentTime}
				messageQueue <- message
			}
		} else if strings.HasPrefix(string(mg), "@") {
			// 私聊(@dt hello world)
			userLoinArr := strings.Split(string(mg), " ")
			toUser := strings.Replace(userLoinArr[0], "@", "", -1)
			// 处理信息
			if toUser != "" {
				// 私聊信息
				message := &message{conUserMap[con], toUser, mt, string(mg), currentTime}
				messageQueue <- message
			}
		} else {
			// 全部用户
			message := &message{conUserMap[con], "all", mt, string(mg), currentTime}
			messageQueue <- message
		}
	}
}

// 处理用户输入
func readMessage() {
	for {
		select {
		case sendMessage := <-messageQueue:
			messageInfo := sendMessage.createTime + " " + sendMessage.fromUser + ":" + sendMessage.content
			fromUserCon, fromUserConOk := userConMap[sendMessage.fromUser]
			toUserCon, toUserConOk := userConMap[sendMessage.toUser]
			if sendMessage.toUser == "all" {
				// 全部用户
				for con := range conUserMap {
					con.WriteMessage(sendMessage.contentType, []byte(messageInfo))
				}
			} else if fromUserConOk || toUserConOk {
				// 私聊
				fromUserCon.WriteMessage(sendMessage.contentType, []byte(messageInfo))
				toUserCon.WriteMessage(sendMessage.contentType, []byte(messageInfo))
			}
		}
	}
}

func main() {
	fmt.Println("123")
	// 处理用户输入
	go readMessage()
	http.HandleFunc("/", home)
	http.HandleFunc("/ws", ws)
	err := http.ListenAndServe("127.0.0.1:8088", nil)
	if err != nil {
		fmt.Println(err)
	}
}
