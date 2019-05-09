# [Hornet](https://github.com/GhostZCH/hornet)

`Hornet`是一个轻量级HTTP缓存引擎。针对现有开源HTTP缓存引擎面对的问题，本着轻、快、集群化和便于二次开发的原则进行开发和设计。

## 背景

在生产环境和测试环境上使用和测试了多种缓存引擎后，发现已有的成品都有些不如人意之处（详见下表）。于是自己动手开发新引擎，希望新的引擎能够“灵巧迅猛，群起而战”，于是便命名为Hornet(黄蜂)。

|引擎|优点|缺点|
|--|--|--|
|proxy-cache(nginx)|与nginx一体，方便在nginx内操作|为每一个key分配一个文件，产生大量小文件，不适合大规模使用|
|varnish| 性能好|重启后会丢失已有的缓存信息，不支持集群|
|squid| http支持较好 | 架构老旧，不支持多核，性能较差，不支持集群 |
|traffic server| 性能高，功能全 | 代码量惊人，没有二次开发的可能性，不支持集群|
|redis/memcached| 性能高 | 仅支持内存缓存，缓存量太小|
|其他支持kv的数据库| 可靠性高，技术成熟 | 需要额外设置过期时间或者容量有限，通常性能不高。大多数据库都考虑读写均衡，但是缓存需要的是极高的读性能写性能要求较低|

## 特性

+ 足够轻，总代码量控制在3k以内，每一个功能都经过反复推敲，力图用最简洁的代码实现最实用的功能。比起大而全，短小精干，便于测试维护才是本程序追求的目的
+ 足够快，希望可以在一个普通的服务器上达到和已有知名程序相同数量级的缓存读取速度，支持多级缓存（mem, ssd, hdd）
+ 便于二次开发，任何程序无法适应企业级的生产环境，与其追加大量代码，不如给出清晰简易的框架，方便使用者添加自己中意的功能
+ 集群部署，在单机部署的基础上实现两种集群部署方式方便具体场景使用，通过udp多播自动组网，无需人工干预
+ 功能新颖，支持多级缓存的缓和和单独使用，支持多种方式的快速缓存删除方式（regex，mask,tag,group等）, 较少的代码意味着可以便利的添加新的功能

## 应用场景

+ 较高的流量和缓存，包含多个域名，有紧急删除的需求
+ 长期运行，偶尔更新，不需要热启动，通过集群和二级缓存提高可靠性
+ 运行在其他负载均衡系统（nginx, haproxy等）之后
+ 使用专用服务器，有足够的内存和CPU资源

## 部署方式

hornet有两种启动模式，cache和proxy。

+ 单机部署，最简易的方式
+ 使用者做负载均衡，多个hornet以cache模式启动，通过udp多播广播自己的服务地址，客户端（例如nginx）根据业务逻辑，控制每个访问具体使用哪个hornet,一般简易使用一致性hash
![](docs/client-balance.png)
+ 代理模式，以proxy和cache模式启动数个hornet，由proxy模式的hornet做负载均衡和资源分布，客户端只有连接上任意一个proxy都可以使用服务。我们建议在proxy外使用lvs进行负载均衡对外提供单一的服务地址方便使用，但是不必须。
![](docs/proxy-balance.png)
+ 当然，以上只是建议，你也可以自己设计更好的部署方式

## API

+ 通过http请求进行缓存的读取，添加和删除功能。
+ 为了方便定位问题，通过`Hornet-Log`头传递一个信息被打印在访问日志中
+ 仅支持HTTP-1.1
+ 默认使用keepalive
+ 处理过程中出现错误直接断开连接并将错误原因写入日志，并不返回客户端

//TODO: 举例子

### Method:Get

获取缓存资源

+ url: /$id
+ headers: Range, If-Not-Match, If-Modified-Since

### Method:Post

添加缓存资源

+ url: /$id
+ headers: Range, Hornet-Group, Hornet-BitMap, Hornet-Tags, Hornet-Rawkey

### Method:Delete

删除缓存资源,有id只删除id对应的资源，没有id执行批量删除

+ url: /[$id]
+ headers: Hornet-Group, Hornet-Ｍask, Hornet-Regex, Hornet-Tags
+ Hornet-Ｍask删除时与post时的Hornet-BitMap做与运算，不为零的删除
+ Hornet-Regex删除时与post时Hornet-Rawkey做正则运算，匹配的删除
+ Hornet-Tags删除时与post时Hornet-Tag时进行对比，相同的删除

### 备注

+ 每个tag是一个不超过65535的整数，默认为0。可以用来表示文件类型，子文件夹，子用户等属性, 协议，子域名。删除时65535为通配符，不检测这个设置

## 配置

### 参数

    Usage of ./hornet:
    -conf string
            conf file path (default "hornet.yaml")
    -mode string
            start mode cache or proxy (default "cache")

### 配置文件

通过启动参数conf配置一个yaml文件作为路径，同时会读取一个通路径下｀local_｀开头文件作为本地配置覆盖主配置，方便大规模部署时添加少了本地配置。例如：conf路径为｀/path/to/conf/hornet.yaml｀, 则会读取一个｀/path/to/conf/local_hornet.yaml｀作为本地配置。

// TODO详细介绍

    # 通用配置
    common.log.path: /tmp/message
    common.log.level: info
    common.accesslog.path: /tmp/access
    common.accesslog.buf: 4096
    common.sock.req.timeout: 3
    common.sock.idle.timeout: 60
    common.http.header.maxlen: 4096
    common.http.body.bufsize: 65536
    common.heartbeat.addr: 224.0.0.100:3300

    # 缓存配置
    cache.mem.meta: /dev/shm/hornet/meta
    cache.mem.path: /dev/shm/hornet/
    cache.mem.cap: 10240000
    cache.mem.blocksize: 1024000

    cache.ssd.meta: /home/ssd/hornet/meta
    cache.ssd.path: /home/ssd/hornet/
    cache.ssd.cap: 10240000
    cache.ssd.blocksize: 1024000

    cache.hdd.meta: /home/hdd/hornet/meta
    cache.hdd.path: /home/hdd/hornet/
    cache.hdd.cap: 10240000
    cache.hdd.blocksize: 1024000

    cache.addr: 127.0.0.1:1100
    cache.heartbeat_ms: 500
    cache.range_block: 262144
    cache.http.header.discard:
        - Host
        - User-Agent
        - Expect
        - Content-Length
        - Hornet-Raw-Key
        - Hornet-Group

    # 代理配置
    proxy.fault_ms: 2000
    proxy.addr: 127.0.0.1:2200
    proxy.keepalive.count: 10

## 优化建议

+ 减少请求头

## 设计

//TODO

### 启动流程

//　重启画
![hornet-start](docs/start-end.png)

### 文件系统

+ 每一级文件缓存，用N个大文件(Block)组成一个FIFO文件系统, 可以在mem,ssd,hdd中配置一到多个缓存级别
+ 内存缓存使用/dev/shm路径下建立文件使用相同的管理逻辑，重启hornet不不要重新加载内存缓存中的数据(重启机器会丢失)
+ 每一级文件缓存，由一个meta文件记录缓存对象信息，启动时加载到内存并删除meta文件，正常退出会重新生成meta文件
+ 写入时追加到最后一个文件末尾，超过文件大小限制的单独缓存文件, 写满一个文件后，打开新文件
+ 删除文件只删除meta中的记录，

## 技术路线

+ 增加有限的回源功能
+ 增加代理功能
+ range缓存的存取
+ 灵活加载http适配器（用于定制协议）
+ 生成统计信息，提供查看接口
+ 更好的缓存替换算法
+ 可编程特性
+ TODO

## Questions & Answers

// TODO 另外建一个页面

### 回源功能

只有向一个固定的地址转发get请求的回源功能。回源功能十分复杂，需要考虑上游地址，协议，握手，证书，负载均衡等诸多问题，实现需要增加大量代码，但是这并不是缓存的主要功能，所以仅仅支持向特定地址转发的功能，由另一个服务(如nginx)完成回源功能。

### 关于tag

### 关于mask

### 为什么不是LRU

LRU使用最广的方法，但是在CDN实际使用中有一定缺陷，比如遇到爬虫时会造成非热点数据刷掉热点数据。根据数据统计，绝大多数的访问是由及少比例的url产生的，这些对象的访问频率非常高，即使用最简单的FIFO过期方式，也不会对命中率产生太大影响，线上环境一般会部署多级缓存，同时穿透两层缓存的概率很低。但是这种FIFO的方法代码简单很多，不需要每次移动数据或者维护队列。例如：一个资源每分钟访问100次，FIFO的过期过程中大概需要2小时，也就是每隔两小时miss一次，命中率大概是 `（100 * 60 * 2 - 1） / 100 * 60 * 2`，是不是使用LRU对命中的影响可以忽略不计。

### 为什么多个文件不是单个文件多个块

+ 单个文件多个块形式不支持在已经有缓存内容的情况下用户调整缓存块的大小
+ 假设现在第n个块已经写满，准备写第n+1个块，但是测试第n+1个块的数据正在被使用，这时可能要用到读写锁等方式才能保证正常。如果用多个文件则没有这个烦恼，可以直接打开新文件并在meta中把最早的一个文件标记成delete状态，等到use数降到0再删除这个文件。

### 为什么不采整个文件做为一个空间，有空位置就存储的方式

会产生一定程度的碎片空间，也不方便过期旧文件

### 为什么批量删除用tag

目前主流的设计有两种批量删除，有的厂商两种都提供：

+ 按照目录删除，Ａ厂和Ｔ厂都支持这种功能，使用trie树实现
+ 按照正则表达式删除，一些老牌的厂商支持这个功能，这个功能比较鸡肋，对于缓存对象比较多的情况需要消耗几十分钟甚至几个小时才能完成一次,在内存中保存完整的url消耗大量的内存。

根据一些线上使用情况，删除操作使用不是很频繁（与访问量对比），批量删除发生的更少。事实上用户需要的更多是根据前缀和后缀进行删除，比如删除`www.myweb.me/news/*.jpg`完整的正则虽然可以完美的实现功能，但是耗时太长。利用trie树虽然会多删除一些但是可以在很短的时间执行完（几秒或几分钟）。本程序目前使用的是一层目录加tag的方式进行删除（经过测试在1000W对象中删除50%的时间不超过１s）。

第一层目录一般是域名，如www.myweb.me，也可以包含目录信息，如:www.myweb.me/news/sport/。
tag可以灵活使用，建议至少选择一个作为文件扩展名（例如: jpg=1,html=2,..）。也可以有更多灵活的用法例如下面的方式:

+ 表示多层子目录，如/2018/03/01/xxx.jpg -> tag1=2018&tag2=3&tag3=1&tag4=１
+ 标志文件大小范围，如：size < 4k = 1, 4k < size < 32k = 2 
+ 标志图片文件分辨率
+ 参数的hash

利用trie树进行目录删除是一个很好的功能，只是要增加一定量的代码，做为本项目的后续改进方向之一。tags有正则和目录删除不具有的优势，后续将继续保留。