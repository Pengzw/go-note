> 1. Redis、Memcache 的应用场景、最佳实践，以及缓存的一致性设计
> 
> 2. MySQL 的表设计、常用优化手段，以及如何解决分布式事务
>
> 参考: https://www.cnblogs.com/chinanetwind/articles/9460820.html

## 分布式缓存
### 缓存选型 - memcache
- memcache提供简单的kv cache存储, value大小不超过1mb
- memcache 使用了slab方式做内存管理, 存在一定的浪费,
    - 如果大量接近item, 建议调整memcache参数来优化每一个slab增长的ratio
    - 也可以通过设置slab_automove & slab_reassign开启memcache的动态/手动move slab
    - 防止某些slab热点导致内存足够的情况下引发LRU
- 大部分情况下, 简单KV推荐使用memcache, 吞吐量相应都足够好
- 内存池很多种设计，可以参考：ngx_pool_t, tcmalloc等

### 缓存选型 - redis
- 丰富的数据类型，支持增量方式修改部分数据。
- redis因为没有使用内存池，所以存在一定的内存碎片，一般会使用jemalloc来优化内存分配，需要编译的时候使用jemalloc库代替glib的malloc使用。

### 缓存选型 - memcache vs redis
- 它们最大的区别其实是redis单线程（新版本双线程），memcache多线程，所以**qps可能两者差异不大，但是吞吐会有很大的差别**，比如大数据value返回的时候，**redis qps会抖动下降的很厉害，因为单线程工作，其他查询进不来（新版本有不少的改善）**。

- 所以建议纯kv都走memcache，比如我们的关系链服务中用了hashs存储双向关系，但是我们也会使用memcache挡一层来避免hgetall导致的吞吐下降问题。

- 也可以memcache+redis双缓存设计。

### 缓存选型 - proxy
- 单机到集群需要缓存代理

### 缓存选型 - 一致性hash
- 平衡性：尽可能分布到所有的缓冲中去
- 单调性：指如果已经有一些内容通过哈希分派到响应的缓冲中，又有新的缓冲区加入到系统中，那么哈希的结果应能够保证原有已分配的内容可以被映射到新的缓冲区中去，而不会被映射到旧的缓冲集合中的其他缓冲区。
- 分散性：相同内容被存储到不同缓冲中去，降低了系统存储的效率，需要尽量降低分散性。
- 负载：哈希算法应能够尽量降低缓冲的负荷。
- 平滑性：缓存服务器的数目平滑改变和缓存对象的平滑改变是一致的。

一致性哈希算法在服务节点太少时，容易因为节点分布不均匀而造成数据倾斜的问题。比如只有a、b两个节点。

这种情况必然容易造成大量数据集中到某一节点上。为了解决这种数据倾斜问题，**一致性哈希算法引入了虚拟节点机制**，即对每一个服务节点计算多个哈希，每个计算结果位置都放置一个此服务节点，称为虚拟节点

知识点：有界负载一致性hash

### 缓存选型 - hash
将数据的某一特征（key）来计算哈希值，并将哈希值与系统中的节点建立映射关系，从而将哈希值不同的数据分布到不同的节点上。

加入或删除一个节点的时候，大量的数据需要移动。

均衡问题：原始数据的特征值分布不均匀，导致大量的数据集中到一个物理节点上；对于可修改的记录数据，单条记录的数据变大。

高级玩法是抽象slot，基于hash的slot 分片，例如redis-cluster

### 缓存选型 - slot
redis-cluster 把16384槽按照节点数量进行平均分配，由节点进行管理。

对每个key按照crc16规则进行hash运算，把hash结果对16383进行取余，把余数发送给redis节点。

需要注意的是：redis-cluster的节点之间会共享消息，每个节点都会知道是哪个节点负责哪个范围内的数据槽

### 缓存模式 - 数据一致性
数据库喝cache同步更新容易出现数据不一致。

模拟mysql slave做数据复制，再把消息投递到mq，保证至少一次消费：
1. 同步操作db；
2. 同步操作cache；
3. 利用job消费消息，重新补偿一次缓存操作（setex）

保证实效性和一致性

### 缓存模式 - 多级缓存
下游服务的缓存数据实效一定要大于上游时间

### 缓存模式 - 热点缓存
- 小表广播，从removeCache提升为localCache
- 主动监控防御预热
- 基础库框架支持热点发现，自动短时的short-live cache
- 多cluster支持
    - 多key设计：使用多副本，减小节点热点问题

### 缓存模式 - 穿透缓存
- singlefly
    - 对关键字进行一致性hash，使其某一个维度的key一定某种某个节点，然后在节点内使用互斥锁，保证归并回源，但是对于批量查询无解；
- 分布式锁（不建议）
    - 设置一个lock key，有且只有一个人成功，并且返回，交由这个人来执行回源操作，其他候选者轮训cache这个lock key，如果不存在去读数据缓存，hit就返回，miss继续抢锁。
- 队列
    - 如果cache miss，交由队列聚合一个key，来load数据回写缓存，对于miss当前请求可以使用singlefly保证回源，如评论架构实现。适合回源加载数据重的任务，比如评论miss只返回第一页，但是需要构建完成评论数据索引。
- lease
    - 通过加入lease机制，可以很好避免着这两个问题，lease时64-bit的token，与客户端请求的key绑定，对于过时设置，在写入时验证lease。

核心思路，只让一个人去取db，不让db被穿透打挂

### 缓存模式 - incast congestion


## 分布式事务

### 分布式事务 - 事务消息
- Transactional outbox
- Polling publisher
- Transaction log tailing
- 2PC Message Queue

事务消息一旦被可靠的持久化, 我们整个分布式事务, 变为了最终一致性, 消息的消费才能保障最终业务数据的完整性, 所以我们要尽最大努力, 把消息送达到下游的业务消费方, 称为: Best Effort. 只有消息被消费, 整个交易才能算是完整完结.


### 分布式事务 - Best Effort
尽最大努力交付, 主要用于在这样一种场景: 不同的服务平台之间的事务性保证.







