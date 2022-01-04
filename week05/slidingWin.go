package main

import (
    "log"
    "sync"
    "time"
    "errors"
)



// 算法思想 将一个大的时间窗口分成多个小窗口，每次大窗口向后滑动一个小窗口，
// 并保证大的窗口内流量不会超出最大值，
// 这种实现比固定窗口的流量曲线更加平滑。

// 窗口计数器
// 表示每个窗口的请求数
type SlideWindow struct {
    WindowSize      int64   // 每个滑动窗口的大小(秒)
    BucketCount     int64   // 切分窗口的数目大小，每个窗口对应一个桶存储数据。
    CurBucket       int     // 当前的桶
    Limiter         int     // 滑动窗口内限流大小
    Bucket          []int   // 存放每个窗口内的计数
    StartTime       int64   // 开始运行的时间
}

func InitSlideWindow(size int64, limiter int, bucketCount int64) (*SlideWindow, error) {
    if size <= 0 || limiter <= 0 || bucketCount <= 0 {
        return nil, errors.New("Parameter Error")
    }
    return &SlideWindow{
        WindowSize:     size,
        Limiter:        limiter,
        BucketCount:    bucketCount,
        CurBucket:  0,
        Bucket:         make([]int, bucketCount),
        StartTime:      time.Now().Unix(),
    }, nil
}

func (c *SlideWindow) GetCurrTime() int64 {
    return time.Now().Unix()
}

func (c *SlideWindow) GetCurrBucket(curTime int64) int {
    return int(curTime / (c.WindowSize / c.BucketCount) % c.BucketCount)
}

func (c *SlideWindow) Wait() error {
    curTime         := c.GetCurrTime()

    // 计算请求时间是否大于当前滑动窗口的最大时间
    // 算出当前时间和开始时间减去窗口大小的值，作用为计算超出当前滑动窗口的时间
    t               := curTime - c.WindowSize - c.StartTime
    // 如果小于或等于0则代表未超出当前滑动窗口的时间。
    if t < 0 {
        t = 0
    }
    // 用t除以滑动窗口的份数，计算出需要滑动的数量。
    windowsNum      := t / (c.WindowSize / c.BucketCount)
    log.Println("windowsNum", windowsNum)
    c.slide(windowsNum)
    countLimiter    := 0
    for i := 0; i < int(c.BucketCount); i++ {
        countLimiter += c.Bucket[i]
    }
    
    log.Printf("当前滑动窗口countLimiter: %d, Limiter: %d", countLimiter, c.Limiter)
    if countLimiter > c.Limiter {
        return errors.New("Rate Limited") // 开始限流
    }

    index := c.GetCurrBucket(curTime)
    log.Println("当前的bucket", index)
    c.Bucket[index]++
    return nil
}


func (c *SlideWindow) slide(windowsNum int64)  {
    slideNum        := c.BucketCount   // 滑动窗口默认设置为 BucketCount
    if windowsNum == 0 {
        return
    }
    // 如果windowsNum小于BucketCount，则将slideNum设置为windowsNum
    // 如果windowsNum大于BucketCount，就代表需要滑动的窗口大于一轮了，所以直接清空当前所有滑动窗口
    if windowsNum < c.BucketCount {
        slideNum    = windowsNum
    }

    for i := 0; i < int(slideNum); i++ {
        // 根据BucketCount取余，获取当前的bucket
        c.CurBucket = (c.CurBucket +1) % int(c.BucketCount)
        c.Bucket[c.CurBucket] = 0
    }
    c.StartTime = c.StartTime + windowsNum * (c.WindowSize / c.BucketCount)
}

func main()  {
    rate, err       := InitSlideWindow(5, 6, 5)
    if err != nil {
        log.Printf("Error: %s \n", err)
        return
    }
    wg              := sync.WaitGroup{}
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 200; i++ {
            if err:=rate.Wait(); err!=nil {
                log.Printf("request: %s \n", err)
                // do something
            }
            time.Sleep(time.Millisecond * 333)
        }
    }()
    wg.Wait()

}



