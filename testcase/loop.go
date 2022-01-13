package main

import "fmt"
import "time"
import "os"

func main() {
	for {
		fmt.Println("pid:", os.Getpid())
		time.Sleep(time.Second)
	}
}
