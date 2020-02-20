package main

import (
  "os/exec"
  "net"
  "bufio"
  "fmt"
)
/*
Simple notifier that works alongside the honeypot. If enabled, will send
desktop notifications when a connection is made. Works on Ubuntu or systems
with the notify-send command.

I built this to run on my desktop computer, while the honeypot runs on my pi.
*/
func main(){
  listener, _ := net.Listen("tcp", ":3344")
  for {
    conn, _ := listener.Accept()
    reader := bufio.NewReader(conn)
    message, err := reader.ReadString('\n')
    if err == nil {
      notifycmd := exec.Command("notify-send", "Network Notifier", message)
      notifycmd.Run()
    } else {
      fmt.Println(err.Error())
    }
    conn.Close()
  }
}
