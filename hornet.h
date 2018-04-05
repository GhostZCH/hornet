#pragma once

#include <map>
#include <list>
#include <string>
#include <thread>
#include <limits>
#include <string>
#include <memory>
#include <vector>
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
#include <signal.h>
#include <sys/types.h>
#include <sys/epoll.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netinet/in.h>


using namespace std;


const int BUF_SIZE = 4096;
const int ETAG_LIMIT = 64;
const int EPOLL_WAIT_EVENTS = 1024;
const int TAG_LIMIT = 4;
const size_t BLOCK_SIZE = 128 * 1024 * 1024;
const size_t ITEM_LIMIT = BLOCK_SIZE;


#include "item.h"
#include "disk.h"
#include "request.h"
#include "event.h"
#include "handlers.h"
#include "process.h"
