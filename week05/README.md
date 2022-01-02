```
// 限流器结构体
type Limiter
	// 构建一个限流器，r 是每秒放入的令牌数量，b 是桶的大小
    func NewLimiter(r Limit, b int) *Limiter

	// 分别返回 b 和 r 的值
    func (lim *Limiter) Burst() int
    func (lim *Limiter) Limit() Limit

	// token 消费方法
    func (lim *Limiter) Allow() bool
    func (lim *Limiter) AllowN(now time.Time, n int) bool
	func (lim *Limiter) Reserve() *Reservation
    func (lim *Limiter) ReserveN(now time.Time, n int) *Reservation
    func (lim *Limiter) Wait(ctx context.Context) (err error)
    func (lim *Limiter) WaitN(ctx context.Context, n int) (err error)

	// 动态流控
    func (lim *Limiter) SetBurst(newBurst int)
    func (lim *Limiter) SetBurstAt(now time.Time, newBurst int)
    func (lim *Limiter) SetLimit(newLimit Limit)
    func (lim *Limiter) SetLimitAt(now time.Time, newLimit Limit)
    
```