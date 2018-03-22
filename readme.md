# Hornet

Hornet是一个用C++开发针对CDN的轻量级缓存引擎。

## 背景

在生产环境和测试环境上使用和测试了数种缓存引擎后，发现已有的成品都有一些不能满足需求的地方（见下方表格），于是诞生了开发本程序的想法。

|引擎|优点|缺点|
|--|--|--|
|proxy-cache(nginx)|与nginx一体，方便在nginx内操作|为每一个key分配一个文件，产生大量小文件，不适合大规模使用|
|varnish| 性能好|重启后会丢失已有的缓存信息|
|squid| http支持较好 | 架构老旧，不支持多核，性能较差 |
|traffic server| 性能高，功能全 | 代码量惊人，没有二次开发的可能性|
|redis/memcached| 性能高 | 仅支持内存缓存，缓存量太小|
|其他支持kv的数据库| - | 需要额外设置过期时间或者容量有限，或者性能不高（大多数据库都考虑读写均衡，但是缓存需要的是极高的读性能写性能要求较低）|

## 原则 & 功能

整个程序设计的原则是高效，可靠，轻量级，方便二次开发。每一个功能都经过反复推敲， 力图用最简洁的代码实现最实用的功能，每一个实现方案都要在性能功能和实现复杂度做权衡，拒绝为少量性能功能提升增加大量的代码。丰富的功能往往意味着繁杂的配置和大量多余的功能，本引擎的目标则是提供一个便于二次定制化开发的基础，让每个用户都能在维护少量代码的基础上实现较好的定制化服务。

+ 通过HTTP协议与其他程序通信
+ 仅支持带有epoll，accept4, 等特性的linux系统，（ubuntu 14.04作为标准环境）
+ 支持内存缓存和硬盘缓存，数据在重启后仍然能够正常使用
+ 设计容量可以存储千万级缓存对象，每个缓存对象可设置一个文件夹路径（HASH）和4个Tag
+ 可以删除其中的一个，或根据文件夹和tag删除一批（秒级删除）
+ 不支持回源功能
+ 不检测恶意攻击和非正常使用
+ 用几个文件存储大量的小文件
+ 通过expire选项控制过期时间
+ 当缓存满时自动删除一部分旧数据
+ 在一个正常的服务器上，qps不低于30k
+ 分别输出access和error日志
+ 支持操作系统的信号量，退出，截断日志
+ PUT一定次数才缓存
+ 提供一个额外的日志头，引擎会将这个头部加入到日志了方便定位问题（一般是url）
+ 返回插入一个头部，用于标示缓存状态，一般是主机名和age

## 应用场景

+ 较高的流量和缓存，包含多个域名，有紧急删除的需求
+ 长期运行，偶尔更新，不需要热启动
+ 通过集群和二级缓存提高可靠性
+ 运行在其他负载均衡系统（nginx, haproxy等）之后
+ 使用专用服务器有足够的内存和CPU资源

## API

+ 每个tag是一个不超过65535的整数(可以用来表示文件类型，子文件夹，子用户等属性, 协议，子域名)，65535为通配符（ffff），不检测这个设置，用不到可以将tag设置为固定值，传入用固定长度为4的16进制数字表示如（0a23），可以多个tag当做一个使用

* GET /$dir/$id?tag1=$tag1&$tag2=$tag2 HTTP/1.1

    例如：

        GET /1deab4b3bda9d0721b30c6a63e427eab/d41d8cd98f00b204e9800998ecf8427e?tag1=001e&tag2=ffff
        log-header: www.myweb.com/2018/02/xxx.jpg?x=480&y=360

* PUT /$dir/$id?tag1=$tag1&$tag2=$tag2 HTTP/1.1

	例如：

        PUT /1deab4b3bda9d0721b30c6a63e427eab/d41d8cd98f00b204e9800998ecf8427e?tag1=001e&tag2=0000
        log-header: www.myweb.com/2018/02/xxx.jpg?x=480&y=360


* DEL /$dir/$id?tag1=$tag1&$tag2=$tag2 HTTP/1.1

	例如：

        DEL /1deab4b3bda9d0721b30c6a63e427eab/d41d8cd98f00b204e9800998ecf8427e?tag1=001E&tag2=0000 HTTP/1.1
        log-header: www.myweb.com/2018/02/xxx.jpg?x=480&y=360

        DEL /1deab4b3bda9d0721b30c6a63e427eab/ffffffffffffffffffffffffffffffff?tag1=001E&tag2=0000 HTTP/1.1
        log-header: www.myweb.com/*.jpg

        DEL /1deab4b3bda9d0721b30c6a63e427eab/ffffffffffffffffffffffffffffffff?tag1=ffff&tag2=0000 HTTP/1.1
        log-header: www.myweb.com/*

## 设计

### 架构

master-woker

### 文件系统

disk

环状缓存block = 128m
超过限制的单独缓存文件（TODO）

### 流程

#### 访问流程

#### 启动流程

#### 缓存过期

## Q & A

### 为什么不是LRU

### 为什么不增加回源功能

## 测试结果

## TODO

+ 完成文档
+ 架构图
+ 流程图
+ Disk代码
+ 大文件单独处理
+ http代码
+ 测试

            // we can delete some data file manully if needed
