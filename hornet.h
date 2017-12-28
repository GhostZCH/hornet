#pragma once

#include <map>
#include <list>
#include <thread>
#include <limits>
#include <string>
#include <memory>
#include <iostream>
#include <functional>
#include <unordered_map>

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
typedef struct epoll_event Event;
typedef struct sockaddr_in Address;

// only allow GET PUT or DEL so the length of request line is fixed
const int BUF_SIZE = 4096;

const short METHOD_GET = 'G'; 
const short METHOD_PUT = 'P';
const short METHOD_DEL = 'D';

const int EPOLL_WAIT_EVENTS = 1024;
socklen_t ADDR_SIZE = sizeof(Address);


#include "record.h"
#include "device.h"
#include "request.h"
#include "event.h"
#include "worker.h"
#include "master.h"

extern Device g_device;
extern unordered_map<int, Request> g_requests_map;
