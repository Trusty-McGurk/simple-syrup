package main

import (
  "net"
  "time"
   "os"
  "io/ioutil"
  "bufio"
  "fmt"
  "sync"
  ssh "golang.org/x/crypto/ssh"
  //term "golang.org/x/crypto/ssh/terminal"
)

const(
  maxConnections = 5
  notifier = false
  notifieraddr = "192.168.1.2:3344" //ip address of the network notifier, if you wanna do that
  loginlog = "logins.log"
)

func handle(sshconn *(ssh.ServerConn), newchans <-chan ssh.NewChannel, req <-chan *(ssh.Request), done chan int){
  go handleRequests(req)
  fmt.Printf("Connection from %s upgraded to SSH.\n", sshconn.RemoteAddr().String())
  if notifier {
    notif := fmt.Sprintf("Connection from %s upgraded to SSH.", sshconn.RemoteAddr().String())
    go SendNotification(notif)
  }
  handleChannels(newchans, sshconn)

  sshconn.Close()
  done <- 1
  fmt.Println("Connection closed.")
  if notifier {
    SendNotification("SSH connection closed.")
  }
}

func handleRequests(req <-chan *(ssh.Request)){
  for request := range req {
    fmt.Printf("Received request: %+v\n", request)
  }
}

func handleChannels(newchans <-chan ssh.NewChannel, sshconn *(ssh.ServerConn)) {
  for nchan := range newchans {
    if t := nchan.ChannelType(); t != "session" {
			nchan.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
			continue
		}

    channel, requests, _ := nchan.Accept()
    var wg1 sync.WaitGroup
    wg1.Add(1)
    go func(requests <-chan *(ssh.Request)) {
      defer wg1.Done()
      var wg2 sync.WaitGroup
      for req := range requests {
        if req.Type == "shell" {
          req.Reply(true, nil)
          wg2.Add(1)

          go handleAndLogSshConn(sshconn, channel, wg2)

        }
      }
      wg2.Wait()
    }(requests)
    wg1.Wait()
  }
}

func handleAndLogSshConn(sshconn *(ssh.ServerConn), channel ssh.Channel, wg sync.WaitGroup){
  defer wg.Done()
  t := time.Now()
  filename := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.log", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
  f, _ := os.Create(filename)

  address := sshconn.RemoteAddr().String()
  f.WriteString(address)
  f.WriteString("\n\n")
  bytesread := 0
  buffer := make([]byte, 1024)
  for {
    l, err := channel.Read(buffer)
    bytesread += l
    if err != nil{
      f.WriteString(fmt.Sprintf("\n\nConnection closed. Reason: %s", err.Error()))
      break;
    }
    f.Write(buffer[:l])
    if bytesread >= 5000 {//let's not accept more than 5kb
      f.WriteString("\n\nConnection closed. Reason: Client tried to write more than 5kb")
      break;
    }
  }
}

func Password(mdata ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
  return nil, nil
}

func LogLogins(mdata ssh.ConnMetadata, method string, err error){
  f, _ := os.Open(loginlog)
  f.WriteString(mdata.RemoteAddr().String())
  f.WriteString("\n")
  f.WriteString(method)
  f.WriteString("\n\n")
  f.Close()
}

func SendNotification(message string) {
  conn, err := net.Dial("tcp", notifieraddr)
  if err != nil {
    fmt.Println("Error sending notification: Could not connect to notification server.")
    return
  }
  writer := bufio.NewWriter(conn)
  writer.WriteString(message)
  writer.WriteString("\n")
  writer.Flush()
  conn.Close()
}

func main(){
  threads := 0
  done := make(chan int)
  listener, _ := net.Listen("tcp", ":22")
  privatekeybytes, _ := ioutil.ReadFile("./id_rsa")
  privatekey, _ := ssh.ParsePrivateKey(privatekeybytes)
  config := ssh.ServerConfig{
    //PasswordCallback: Password,
    NoClientAuth: true,//no security no problem
    ServerVersion: "SSH-2.0-OpenSSH_7.9p1 Raspbian-10+deb10u1",
    AuthLogCallback: LogLogins,
  }

  config.AddHostKey(privatekey)

  //Keeps track of how many connections are open at once
  go func(done chan int, threads *int){
    for {
      <- done
      if *threads > 0 {
        (*threads)--
      }
    }
  }(done, &threads)

  for {
    conn, _ := listener.Accept()
    fmt.Printf("New connection form %s\n", conn.RemoteAddr().String())
    if notifier {
      notif := fmt.Sprintf("New connection from %s.", conn.RemoteAddr().String())
      go SendNotification(notif)
    }

    if threads >= maxConnections { //let's not have more than 5 connections open at once
      conn.Close()
      continue
    }

    threads++

    sshconn, nchan, req, err := ssh.NewServerConn(conn, &config)
    if err != nil {
      fmt.Print("Error in upgrading connection to SSH: ")
      fmt.Println(err.Error())
      if notifier {
        SendNotification(fmt.Sprintf("Connection to %s closed. Error upgrading connection to SSH.", conn.RemoteAddr().String()))
      }
      done <- 1
      conn.Close()
    } else {
      go handle(sshconn, nchan, req, done)
    }
  }
}
