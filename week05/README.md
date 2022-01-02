> 1. 微服务的隔离实现，以及架构设计中的隔离实现
> 
> 2. 进程内超时控制和跨进程超时控制
> 
> 3. 程序自保护避免过载，抛弃一定的流量完成自适应限流
> 
> 4. 单机限流、多租户场景的分布式限流
> 
> 5. 节点故障的容错逻辑、重试容错的策略和设计


## 隔离
### 服务隔离
隔离设计源于船舶行业，一般而言无论大船还是小船，都会有一些隔板，将船分为不同的空间，这样如果有船舱漏水一般只会影响这一小块空间，不至于把整船沉默。
#### 服务隔离
- 动静分离
    - 小到CPU的cacheline falsesharing、数据库表的动静表， 大到架构设计中的图片、静态资源等cdn缓存加速
- 读写分离
    - 主从、Replicaset、CQRS

#### 轻重隔离
- 核心隔离
    - 核心业务独立部署，非核心业务共享资源
- 快慢隔离
    - 服务的吞吐量
    - 服务的实时需求
- 热点隔离
    - remote cache 到 local cache
    - 集群 or ..

#### 物理隔离
- 线程
    - 常见的例子就是线程池,Golang 中一般不用过多考虑
- 进程
    - 使用容器化服务，跑在 k8s 上这就是一种进程级别的隔离
- 集群
    - 非常重要的服务我们可以部署多套，在物理上进行隔离，常见的有异地部署，也可能就部署在同一个区域
    - grpc
- 机房
    - 服务的不同副本尽量的分配在不同的可用区，实际上就是云厂商的不同机房，避免机房停电或者着火之类的影响


## 超时
- 让请求尽快消化，控制请求的生命周期
- 网络传递是不可控的
- 请求需要超时配置，默认值不保守策略
- 服务接口提供者，在文档中需要定义超时时间，让服务在合理范围内消化

## 过载保护和限流
### 过载保护（90%的问题）
计算系统临近过载时的峰值吞吐作为限流的阀值来进行流量控制或卸载流量，达到系统保护

#### 令牌桶算法
是一个存放固定容量令牌的桶,按照固定速率往桶里添加令牌.
- 假设限制2r/s, 则按照500毫秒的固定速率往桶中添加令牌
- 同种最多存放b个令牌,满则丢弃
- 当一个n个字节大小的数据包大盗,将从桶中删除n个令牌,接着数据包被发送到网络中
- 如果桶中的令牌不足n个,则数据包将被限流(要么丢弃,要么缓冲区等待)
- Go官方的令牌桶的实现: https://pkg.go.dev/golang.org/x/time/rate

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
##### 初始化令牌桶
直接调用 NewLimiter(r Limit, b int)  即可， r  表示每秒产生 token 的速度， b  表示桶的大小

##### Token 消费
总共有三种 token 消费的方式，最常用的是使用 Wait  阻塞等待
- Allow
- Reserve
- Wait

##### 动态流控
通过调用 SetBurst  和 SetLimit  可以动态的设置桶的大小和 token 生产速率，其中 SetBurstAt  和 SetLimitAt  会将传入的时间 now  设置为流控最后的更新时间
```
func (lim *Limiter) SetBurst(newBurst int)
func (lim *Limiter) SetBurstAt(now time.Time, newBurst int)
func (lim *Limiter) SetLimit(newLimit Limit)
func (lim *Limiter) SetLimitAt(now time.Time, newLimit Limit)
```
##### 实现 基于 ip 的 gin 限流中间件

```
func main() {
	e := gin.Default()
	// 新建一个限速器，允许突发 10 个并发，限速 3rps，超过 500ms 就不再等待
	e.Use(NewLimiter(3, 10, 500*time.Millisecond))
	e.GET("ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	e.Run(":8080")
}
func NewLimiter(r rate.Limit, b int, t time.Duration) gin.HandlerFunc {
	limiters := &sync.Map{}

	return func(c *gin.Context) {
		// 获取限速器
		// key 除了 ip 之外也可以是其他的，例如 header，user name 等
		key := c.ClientIP()
		l, _ := limiters.LoadOrStore(key, rate.NewLimiter(r, b))

		// 这里注意不要直接使用 gin 的 context 默认是没有超时时间的
		ctx, cancel := context.WithTimeout(c, t)
		defer cancel()

		if err := l.(*rate.Limiter).Wait(ctx); err != nil {
			// 这里先不处理日志了，如果返回错误就直接 429
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": err})
		}
		c.Next()
	}
}
```
##### go-stress-testing 压测

```
 ./go-stress-testing-mac -c 20 -n 1 -u http://127.0.0.1:8080/ping

开始启动  并发数:20 请求数:1 请求参数:

─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────
  耗时│  并发数│  成功数│  失败数│   qps  │ 最长耗时│  最短耗时│ 平均耗时│  下载字节│ 字节每秒│ 错误码
─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────
   1s│     20│     11│      9│   63.79│  438.48│   45.37│  313.53│     152│     259│200:11;429:9


*************************  结果 stat  ****************************
处理协程数量: 20
请求总数（并发数*请求数 -c * -n）: 20 总请求时间: 0.586 秒 successNum: 11 failureNum: 9
*************************  结果 end   ****************************
```
可以发现总共成功了 11 个请求，失败了 9 个，这是因为我们桶的大小是 10 所以前 10 个请求都很快就结束了，第 11 个请求等待 333.3 ms 就可以完成，小于超时时间 500ms，所以可以放行，但是后面的请求确是等不了了，所以就都失败了，并且可以看到最后一个成功的请求的耗时为 336.83591ms  而其他的请求耗时都很短

#### 漏桶算法
作为计量工作时，可以用于流量整形和流量控制，漏桶算法如下：
- 一个固定容量的漏桶，按照常量固定速率流出水滴
- 如果桶空，则不需要流出水滴
- 可以以任意速率流入水滴到漏桶
- 如果流入水滴超出了桶的容量，则流入的水滴丢弃，而漏桶容量不变
- https://pkg.go.dev/go.uber.org/ratelimit

#### 漏斗桶和令牌桶的缺陷
漏斗桶和令牌桶能保护系统不被拖垮，其防护思路都是设定一个指标。超过负载则减少流量，负载降低则恢复流量进入。但都是被动的，实际效果取决于限流阀值设置是否合理，而合理性往往不容易。
- 集群增加机器或者减少机器限流是否要重制
- 设置限流阀值的依据是什么
- 人力运维成本是否过高
- 当用户反馈时，其实已经错过高峰期

#### 基于cpu计算的过载保护
##### 原理
- cpu：实用一个独立的线程采样，每隔250ms触发一次。在计算均值时，实用了简单滑动平均去除峰值的影响。
- inflight：当前服务中正在进行的请求的数量
- pass&RT：最近5s，pass为每个100ms采样窗口内成功请求的数量，rt为每个采样窗口中平均响应时间

##### 设计
- 实用cpu的滑动均值（cpu>80%）作为启发阀值，一旦触发进入到过载保护阶段，算法：(pass*rt) < inflight
- 限流效果生效后，持续冷却5s
- 冷却后重新检测

### 限流（兜底最后10%的问题）
限流指在一段时间内，定义某个服务可以接收或处理多少个请求的技术。例如，通过限流，可以过滤产生流量峰值的客户和微服务，或者可以确保应用程序在自动扩展失效前不会出现过载的情况。
#### 限流的常态问题 
- 令牌桶、漏桶针对单个节点，无法分布式限流
- QPS限流
    - 不同的请求可能需要数量迥异的资源来处理
    - 某种静态QPS限流不是特别准
- 给每个租户设置限流
    - 全局过载发生时候，针对某些“异常”进行控制
    - 一定程度的“超卖”配额
- 按照优先级丢弃
- 拒绝请求也需要成本

### 限流（分布式）
分布式限流，是为了控制某个应用全局的流量，而非针对单个节点纬度
- redis版分布式
    - 单个大流量的接口，使用redis容易产生热点
    - pre-request模式对性能有一定影响，高频的网络往返
- 异步批量获取quota
    - 每次心跳后，异步批量获取quota，可以减少redis的频次，获取完以后本地消费，基于令牌桶拦截
    - 初次使用默认值，之后可以使用产生的历史窗口数据，进行quota请求
- 最大化合理分配quota资源
    - 分享技术“最大最小公平分享”
        - 优先分配资源需求小的服务

类型 | 优点 | 缺点 | 现有实现
---|---|---|---
单机限制|实现简单；稳定可靠；性能高|流量不均匀会引发限制；机器数变化时配额要人工调整，容易出错|golang/x/time/ratelimit
动态流控|根据服务情况动态限流；不用调整配额|需要主动搜集请求的性能数据（cpu、load、成功率、耗时）；客户端主动善意限流；一般只限于接口调用，支持范围小，应用场景狭窄|go-kratos的bbr；广义上各种连接池
全局限流|流量不均不会误触发限流；机器数变动时无需调整；应用场景丰富，接口db等任何资源都可以使用|实现较复杂；需要手动配置|无

### 限流 - 按重要性限流
- 最重要：拒绝服务请求会造成非常严重的用户可见的问题
- 重要：拒绝服务请求会造成用户可见的问题，但可能没那么严重
- 可丢弃：可以容忍某种程度的不可用性，可以过几分钟、几小时后重试的服务
- 完全可丢弃：偶尔会完全不可用

### 熔断
为了限制操作的持续时间，我们可以使用超时，超时可以防止挂起操作并保证系统可以响应。因为处于高度动态的环境中，几乎不可能确定在每种情况下都能正常工作的准确的时间的限制。

典型场景：服务依赖的资源出现大量错误。

### 限流 = 客户端流控
用户总是积极重试，访问一个不可达的服务
- 客户端需要限制请求频次

## 降级&重试


## 重试和负载均衡


