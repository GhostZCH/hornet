## hornet

暂时只是demo

## 性能测试

20190210

    ./wrk http://127.0.0.1:8080/1/2 -c 1000 -d 20s -t 3
    Running 20s test @ http://127.0.0.1:8080/1/2
    3 threads and 1000 connections
    Thread Stats   Avg      Stdev     Max   +/- Stdev
        Latency    32.20ms    8.00ms 207.40ms   82.07%
        Req/Sec    10.28k     1.70k   15.91k    73.37%
    611134 requests in 20.03s, 83.93MB read
    Requests/sec:  30507.68
    Transfer/sec:      4.19MB

    curl -vv http://127.0.0.1:8080/1/2
    *   Trying 127.0.0.1...
    * TCP_NODELAY set
    * Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
    > GET /1/2 HTTP/1.1
    > Host: 127.0.0.1:8080
    > User-Agent: curl/7.58.0
    > Accept: */*
    > 
    < HTTP/1.1 200 OK
    < Server: Hornet
    < Connection: keep-alive
    < Accept: */*
    < Content-Length: 4
    < Content-Type: application/x-www-form-urlencoded
    < 
    * Connection #0 to host 127.0.0.1 left intact
    1111

## TODO

- [x] go 调查，基础功能测试
- [x] go 性能测试
- [x] 搭建原型基础框架
- [x] 读取配置
- [x] store加入
- [x] http功能封装(50％)
- [] 日志输出 (50％)
- [] 性能测试
- [] 基础功测试 (DEL 没测试)
- [] 回源功能



