

## 作业内容：
1. 使用 redis benchmark 工具, 测试 10 20 50 100 200 1k 5k 字节 value 大小，redis get set 性能。
2. 写入一定量的 kv 数据, 根据数据大小 1w-50w 自己评估, 结合写入前后的 info memory 信息 , 分析上述不同 value 大小下，平均每个 key 的占用内存空间。

### 测试工具
Redis客户端源码包含一个名为redis-benchmark的性能测试工具，它可以模拟N个客户端同时向Redis发送M条查询命令的应用场景。

redis-benchmark工具的使用方法如下所示：
```
redis-benchmark -h {IP} -p {Port} -a {password} -n {nreqs} -r {randomkeys} -c {connect_number} -d {datasize} -t {command}

参数参考值：-c {connect_number}：200，-n {nreqs}：10000000，-r {randomkeys}：1000000，-d {datasize}：32。

-t 表示基准测试使用的线程数量
-c 表示客户端连接数
-d 表示单条数据大小，单位byte
-n 表示测试包数量
-r 表示使用随机key数量

其中参数-h的ip取Redis实例IP值，当实例为Cluster集群实例时，其ip为任意一个实例IP的值。
```


#### 10 字节压测(10w次请求)

```
[root@centos7 bin]# ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 100000 -t get,set -r 1000000 -d 10
====== SET ======
  100000 requests completed in 1.23 seconds
  200 parallel clients
  10 bytes payload
  keep alive: 1

29.31% <= 1 milliseconds
100.00% <= 4 milliseconds
81499.59 requests per second

====== GET ======
  100000 requests completed in 1.34 seconds
  200 parallel clients
  10 bytes payload
  keep alive: 1

26.83% <= 1 milliseconds
100.00% <= 4 milliseconds
74626.87 requests per second

```


#### 10 字节压测(100w次请求)
```
[root@centos7 bin]# ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 1000000 -t get,set -r 1000000 -d 10
====== SET ======
  1000000 requests completed in 12.73 seconds
  200 parallel clients
  10 bytes payload
  keep alive: 1

30.57% <= 1 milliseconds
100.00% <= 6 milliseconds
78523.76 requests per second

====== GET ======
  1000000 requests completed in 11.89 seconds
  200 parallel clients
  10 bytes payload
  keep alive: 1

36.02% <= 1 milliseconds
100.00% <= 13 milliseconds
84132.59 requests per second

```


#### 1k字节压测(10w次请求)
```
[root@centos7 bin]# ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 100000 -t get,set -r 1000000 -d 1024
====== SET ======
  100000 requests completed in 1.32 seconds
  200 parallel clients
  1024 bytes payload
  keep alive: 1

19.70% <= 1 milliseconds
100.00% <= 5 milliseconds
75757.57 requests per second

====== GET ======
  100000 requests completed in 1.27 seconds
  200 parallel clients
  1024 bytes payload
  keep alive: 1

31.44% <= 1 milliseconds
100.00% <= 3 milliseconds
78554.59 requests per second
```

#### 5k字节压测(10w次请求)
```
[root@centos7 bin]# ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 100000 -t get,set -r 1000000 -d 5120
====== SET ======
  100000 requests completed in 2.30 seconds
  200 parallel clients
  5120 bytes payload
  keep alive: 1

0.10% <= 1 milliseconds
100.00% <= 42 milliseconds
43440.48 requests per second

====== GET ======
  100000 requests completed in 1.27 seconds
  200 parallel clients
  5120 bytes payload
  keep alive: 1

11.12% <= 1 milliseconds
100.00% <= 4 milliseconds
78926.60 requests per second

```

#### 压测总结
经压测得, redis 读写基本能维持在每秒7~8s的并发数. 但是当key的数据量越大超过1k后, 很明显能看到redis的吞吐量急速下滑


### info 查看

#### flushdb 之后
```
# Memory
used_memory:1412792
used_memory_human:1.35M
used_memory_rss:217501696
used_memory_rss_human:207.43M
used_memory_peak:802954448
used_memory_peak_human:765.76M
used_memory_peak_perc:0.18%
used_memory_overhead:1291564
used_memory_startup:791408
used_memory_dataset:121228
used_memory_dataset_perc:19.51%
allocator_allocated:1732864
allocator_active:8593408
allocator_resident:213352448
total_system_memory:3973226496
total_system_memory_human:3.70G
used_memory_lua:37888
used_memory_lua_human:37.00K
used_memory_scripts:0
used_memory_scripts_human:0B
number_of_cached_scripts:0
maxmemory:0
maxmemory_human:0B
maxmemory_policy:noeviction
allocator_frag_ratio:4.96
allocator_frag_bytes:6860544
allocator_rss_ratio:24.83
allocator_rss_bytes:204759040
rss_overhead_ratio:1.02
rss_overhead_bytes:4149248
mem_fragmentation_ratio:137.96
mem_fragmentation_bytes:215925160
mem_not_counted_for_evict:0
mem_replication_backlog:0
mem_clients_slaves:0
mem_clients_normal:500156
mem_aof_buffer:0
mem_allocator:jemalloc-5.1.0
active_defrag_running:0
lazyfree_pending_objects:0

```

#### 插入10w条1k
- 用命令行插入1w数据
    - ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 100000 -t set -r 10000 -d 1024
    - ./redis-benchmark -h 127.0.0.1 -p 6379 -c 200 -n 100000 -t set -r 1000 -d 10
```
used_memory:14738808
used_memory_human:14.06M
used_memory_rss:82284544
used_memory_rss_human:78.47M
used_memory_peak:802954448
used_memory_peak_human:765.76M
used_memory_peak_perc:1.84%
used_memory_overhead:1626004
used_memory_startup:791408
used_memory_dataset:13112804
used_memory_dataset_perc:94.02%
allocator_allocated:14985248
allocator_active:15425536
allocator_resident:78061568
total_system_memory:3973226496
total_system_memory_human:3.70G
used_memory_lua:37888
used_memory_lua_human:37.00K
used_memory_scripts:0
used_memory_scripts_human:0B
number_of_cached_scripts:0
maxmemory:0
maxmemory_human:0B
maxmemory_policy:noeviction
allocator_frag_ratio:1.03
allocator_frag_bytes:440288
allocator_rss_ratio:5.06
allocator_rss_bytes:62636032
rss_overhead_ratio:1.05
rss_overhead_bytes:4222976
mem_fragmentation_ratio:5.60
mem_fragmentation_bytes:67586752
mem_not_counted_for_evict:0
mem_replication_backlog:0
mem_clients_slaves:0
mem_clients_normal:303524
mem_aof_buffer:0
mem_allocator:jemalloc-5.1.0
active_defrag_running:0
lazyfree_pending_objects:0


```

当redis被用作缓存时，有时我们希望了解key的大小分布，或者想知道哪些key占的空间比较大
### 分析工具
#### bigKeys
这是redis-cli自带的一个命令。对整个redis进行扫描，寻找较大的key
```
./redis-cli --bigkeys

# Scanning the entire keyspace to find biggest keys as well as
# average sizes per key type.  You can use -i 0.1 to sleep 0.1 sec
# per 100 SCAN commands (not usually needed).

[00.00%] Biggest string found so far 'key:000000000586' with 10 bytes
[00.00%] Biggest string found so far 'key:000000005975' with 1024 bytes

-------- summary -------

Sampled 10000 keys in the keyspace!
Total key length in bytes is 160000 (avg len 16.00)

Biggest string found 'key:000000005975' has 1024 bytes

0 lists with 0 items (00.00% of keys, avg size 0.00)
0 hashs with 0 fields (00.00% of keys, avg size 0.00)
10000 strings with 9226000 bytes (100.00% of keys, avg size 922.60)
0 streams with 0 entries (00.00% of keys, avg size 0.00)
0 sets with 0 members (00.00% of keys, avg size 0.00)
0 zsets with 0 members (00.00% of keys, avg size 0.00)


```

说明：
- 该命令使用scan方式对key进行统计，所以使用时无需担心对redis造成阻塞。
- 输出大概分为两部分，summary之上的部分，只是显示了扫描的过程。summary部分给出了每种数据结构中最大的Key。
- 统计出的最大key只有string类型是以字节长度为衡量标准的。list,set,zset等都是以元素个数作为衡量标准，不能说明其占的内存就一定多。所以，如果你的Key主要以string类型存在，这种方法就比较适合。

#### debug object key
redis的命令，可以查看某个key序列化后的长度。

```
127.0.0.1:6379> debug object key:000000005975
Value at:0x7f406d476360 refcount:1 encoding:raw serializedlength:22 lru:15374064 lru_seconds_idle:966

127.0.0.1:6379> get key:000000005975
"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

```

- 关于输出的项的说明：
    - Value at：key的内存地址
    - refcount：引用次数
    - encoding：编码类型
    - serializedlength：序列化长度
    - lru_seconds_idle：空闲时间
- 几个需要注意的问题
    - serializedlength是key序列化后的长度(redis在将key保存为rdb文件时使用了该算法)，并不是key在内存中的真正长度。这就像一个数组在json_encode后的长度与其在内存中的真正长度并不相同。不过，它侧面反应了一个key的长度，可以用于比较两个key的大小。
    - serializedlength会对字串做一些可能的压缩。如果有些字串的压缩比特别高，那么在比较时会出现问题。比如下列

```
127.0.0.1:6379> set str2 abcdefghijklmnopqrstuvwxyz1234
OK
127.0.0.1:6379> debug object str2
Value at:0x7f4077c78428 refcount:1 encoding:embstr serializedlength:31 lru:15375191 lru_seconds_idle:17

```



