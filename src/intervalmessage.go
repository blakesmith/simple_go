package main

import (
	"fmt"
	"time"
)

func intervalMessage(msg string, dur time.Duration) chan string {
	outc := make(chan string)
	go func() {
		for {
			time.Sleep(dur)
			outc <- msg
		}
	}()

	return outc
}

func main() {
	chan1 := intervalMessage("HAHA", 1*time.Second)
	chan2 := intervalMessage("LOLZ", 2*time.Second)

	for {
		select {
		case resp := <- chan1:
			fmt.Println(resp)
		case resp := <- chan2:
			fmt.Println(resp)
		}
	}
}