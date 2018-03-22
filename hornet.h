#pragma once

#include <map>
#include <list>
#include <string>
#include <thread>
#include <limits>
#include <string>
#include <memory>
#include <iostream>
#include <functional>
#include <unordered_map>
#include <unordered_set>


#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <fcntl.h>
#include <errno.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/epoll.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netinet/in.h>

using namespace std;

enum FdType{
    FD_TYPE_NONE,
    FD_TYPE_SERVER,
    FD_TYPE_CLIENT,
    FD_TYPE_WORKER,
    FD_TYPE_MASTER
};


typedef union epoll_data EpollData;
typedef struct epoll_event EpEvent;
typedef struct sockaddr_in Address;

// only allow GET PUT or DEL so the length of request line is fixed
const short METHOD_GET = 'G'; 
const short METHOD_PUT = 'P';
const short METHOD_DEL = 'D';

const int BUF_SIZE = 4096;
const int ETAG_LIMIT = 64;
const int EPOLL_WAIT_EVENTS = 1024;
const int TAG_LIMIT = 4;
const size_t BLOCK_SIZE = 128 * 1024 * 1024;
const size_t ITEM_LIMIT = BLOCK_SIZE;
socklen_t ADDR_SIZE = sizeof(Address);


#include "item.h"
#include "disk.h"
#include "request.h"
#include "event.h"
#include "worker.h"
#include "master.h"
