#pragma once

#include <map>
#include <list>
#include <regex>
#include <mutex>
#include <thread>
#include <memory>
#include <string>
#include <limits>
#include <string>
#include <sstream>
#include <fstream>
#include <vector>
#include <atomic>
#include <iostream>
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

extern const char* VERSION_STR;
extern const int VERSION;

typedef struct sockaddr Address;
const socklen_t ADDR_SIZE = sizeof(Address);

const int EPOLL_WAIT_EVENTS = 1024;

const int ETAG_LIMIT = 64;
const int TAG_LIMIT = 4;

const size_t ACCESS_LOG_BUF = 4096;
const size_t BLOCK_SIZE = 128 * 1024 * 1024;
const size_t ITEM_LIMIT = BLOCK_SIZE;

const int DISK_FD = -1; // fake fd for disk handler

