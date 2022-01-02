package main

import (
    "log"
    "sync"
    "time"
)

// 窗口计数器
// 表示每个窗口的请求数
type WindowCounter struct {
    winSize         int64   // 每个滑动窗口的大小(秒)
    splitNum        int64   // 切分窗口的数目大小，每个窗口对应一个桶存储数据。
    currentBucket   int     // 当前的桶
    Limiter         int     // 滑动窗口内限流大小
    Bucket          []int   // 存放每个窗口内的计数
    StartTime       int64   // 开始运行的时间
}

func New(size int64, limit int, splitNum int64) *WindowCounter {
    return &WindowCounter{
        winSize:        size,
        Limiter:        limit,
        splitNum:       splitNum,
        currentBucket:  0,
        Bucket:         make([]int, splitNum),
        StartTime:      time.Now().Unix(),
    }
}

// return false 开始限流
func (c *WindowCounter) TryLimiter() bool {
    currentTime     := time.Now().Unix()

    // 计算请求时间是否大于当前滑动窗口的最大时间
    // 算出当前时间和开始时间减去窗口大小的值，作用为计算超出当前滑动窗口的时间
    t               := currentTime - c.winSize - c.StartTime
    // 如果小于或等于0则代表未超出当前滑动窗口的时间。
    if t < 0 {
        t = 0
    }
    // 用t除以滑动窗口的份数，计算出需要滑动的数量。
    windowsNum      := t / (c.winSize / c.splitNum)
    c.slideWindow(windowsNum)
    countLimiter    := 0
    for i := 0; i < int(c.splitNum); i++ {
        countLimiter += c.Bucket[i]
    }
    
    log.Printf("当前滑动窗口countLimiter: %d, Limiter: %d", countLimiter, c.Limiter)
    if countLimiter > c.Limiter {
        return false // 开始限流
    }

    index := c.GetCurrentBucket()
    log.Println("当前的bucket", index)
    c.Bucket[index]++
    return true
}

func (c *WindowCounter) GetCurrentBucket() int {
    return int(time.Now().Unix() / (c.winSize / c.splitNum) % c.splitNum)
}

func (c *WindowCounter) slideWindow(windowsNum int64)  {
    slideNum        := c.splitNum   // 滑动窗口默认设置为 splitNum
    if windowsNum == 0 {
        return
    }
    // 如果windowsNum小于splitNum，则将slideNum设置为windowsNum
    // 如果windowsNum大于splitNum，就代表需要滑动的窗口大于一轮了，所以直接清空当前所有滑动窗口
    if windowsNum < c.splitNum {
        slideNum = windowsNum
    }

    log.Println("当前要滑动的窗口数", slideNum)
    for i := 0; i < int(slideNum); i++ {
        // 根据splitNum取余，获取当前的bucket
        
        c.currentBucket = (c.currentBucket +1) % int(c.splitNum)
        log.Printf("当前桶的位置: %d, 大小为: %d",c.currentBucket, c.Bucket[c.currentBucket])
        c.Bucket[c.currentBucket] = 0
    }
    c.StartTime = c.StartTime + windowsNum * (c.winSize / c.splitNum)
}

func main()  {
    c           := New(10, 20, 5)
    c.TryLimiter()

    wg          := sync.WaitGroup{}
    wg.Add(1)
    go func() {
        defer func() {
            wg.Done()
        }()
        for i:=0; i <= 100; i++ {
            for j:=0; j<1; j++ {
                if c.TryLimiter() {

                }
            }
            time.Sleep(time.Millisecond * 100)
        }
    }()
    go func() {
        defer func() {
            wg.Done()
        }()
        for i:=0; i <= 100; i++ {
            for j:=0; j<1; j++ {
                if c.TryLimiter() {

                }
            }
            time.Sleep(time.Millisecond * 500)
        }
    }()
    wg.Wait()

}



