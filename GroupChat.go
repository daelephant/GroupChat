package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

//用户端结构体类型
type Client struct {
	C chan string
	Name string
	Addr string
}

//创建全局map,存储维护在线用户
var onlineMap map[string]Client

//创建全局channel传递用户消息
var message = make(chan string)

func MakeMsg(clnt Client,msg string) (buf string){
	buf = "[" + clnt.Addr +"]" + clnt.Name + ": " + msg
	return
}

func WriteMessageToClient(clnt Client,conn net.Conn){
	//监听用户自带channel上是否有消息
	for msg := range clnt.C{
		conn.Write([]byte(msg + "\n"))
	}
}

func HandConn(conn net.Conn){
	defer conn.Close()
	//创建channel,用来判断用户是否活跃
	hasData := make(chan bool)

	//获取用户网络地址 IP port
	netAddr := conn.RemoteAddr().String()

	//创建新连接用户的结构体 Client  默认用户是IP+port
	clnt := Client{make(chan string), netAddr,netAddr}

	//将新连接的用户，添加到在线用户map，key：IP+port value：client
	onlineMap[netAddr] = clnt

	//创建专门用来给当前用户发送消息的go程

	go WriteMessageToClient(clnt,conn)
    //发送用户上线消息到全集channel中
    message <- MakeMsg(clnt,"login")

    //创建一个channel,用来判断client退出状态

    isQuit := make(chan bool)

    //创建一个匿名的go程，处理用户发送的消息
    go func() {
    	buf := make([]byte, 4096)
		for {
			n,err := conn.Read(buf)
			if n == 0 {
				isQuit <- true
				fmt.Printf("检测到客户端:%s退出\n", clnt.Name)
				return
			}
			if err != nil{
				fmt.Println("conn.Read err:", err)
				return
			}
			//将读到的用户消息，保存到msg中，string类型
			msg := string(buf[:n-1])

			//提取在线用户列表
			if msg == "who" && len(msg) ==3{
				conn.Write([]byte("线上用户列表 :\n"))

				//遍历当前map。获取在线用户量
				for _,user := range onlineMap {
					userInfo := user.Addr + ":" + user.Name+"\n"
					conn.Write([]byte(userInfo))

				}

			} else if len(msg) >=8 && msg[:7] == "rename|"{
				newName := strings.Split(msg,"|")[1]
				clnt.Name = newName
				onlineMap[netAddr] = clnt
				conn.Write([]byte("rename success \n"))
			}else {
				//将读到的用户消息，写入到message中
				message <- MakeMsg(clnt,msg)
			}
			hasData <-true

		}
	}()

    //保证 不退出
    for {
    	//监听channel上的数据流动
		select {
		case <-isQuit:
			delete(onlineMap,clnt.Addr)
			message<- MakeMsg(clnt,"logout")//写入用户退出消息到全局channel
			return
		case <-hasData:
			//do nothing 目的是重置下面的case计时器
		case <-time.After(time.Second * 60):
			delete(onlineMap,clnt.Addr)
			message <- MakeMsg(clnt,"time out leaved")
			return



		}
	}

}


func ManagerMapAndChan(){
	//初始化 onlineMap
	onlineMap = make(map[string]Client)

	//监听全局channel中是否有数据，有数据存储至msg，无数据则阻塞

	for {
		msg := <-message

		//循环发送消息给 在线的全部用户，必须，msg:=<-message 执行完，即读到数据，才解除阻塞

		for _, onlineClient := range onlineMap{
			onlineClient.C <- msg
		}

	}
}

func main(){
	//创建监听socket,设置端口8081
	listener, err := net.Listen("tcp","127.0.0.1:8081")
	if err != nil {
		fmt.Println("Listener err",err)
		return
	}
	defer listener.Close()//记得关闭

	//创建管理者go程，管理map和全局channel
	go ManagerMapAndChan()

	//循环监听客户端数据请求

	for{
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept ERR",err)
			return
		}
		//启动go程处理客户端请求
		go HandConn(conn)
	}

}
