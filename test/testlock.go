package main

import (
	"fmt"
	"sync"
)

// / goroutine 同步锁 和 同步组
func main() {
	var mutex sync.Mutex
	var wait = sync.WaitGroup{}
	var countI = 0

	mutex.Lock()
	fmt.Println("等待执行")
	for i := 0; i < 100; i++ {
		wait.Add(1)
		go func(count int) {
			mutex.Lock()
			countI++
			fmt.Println("输出", countI, count+1)
			mutex.Unlock()
			wait.Done()
		}(i)
	}
	mutex.Unlock()
	wait.Wait()
	fmt.Println("等待执行完成")
}
