#include "request.h"
#include <string.h>

#include <iostream>

using namespace std;


bool RequestHandler::Init(Request *r)
{
    memset(r, 0, sizeof(Request));

    r->fd = -1;
    r->phase = HttpPhase::RECV;
    return true;
}


bool RequestHandler::Read(Request *r)
{
    while (r->process_len < BUF_SIZE) {
        ssize_t n = read(r->fd, r->buffer + r->process_len, BUF_SIZE - r->process_len);
        
        if (n > 0) {
            r->process_len += n;
            continue;
        }

        if (n < 0 && errno == EAGAIN) {
            break;
        }

        return false;
    }

    if (strstr(r->buffer, "\r\n\r\n")) {
        r->process_len = 0;
        r->content_length = 0;
        r->phase = HttpPhase::SEND;

        if (ParseRequestHeader(r) && GenrateResponseHeader(r)) {
            return true;
        }
        return false;
    }

    return true;
}


bool RequestHandler::Write(Request *r)
{
    if (r->phase != HttpPhase::SEND) {
        return false;
    }

    while (r->process_len < r->length) {
        ssize_t n = write(r->fd, r->buffer + r->process_len, r->length - r->process_len);
        if (n > 0) {
            r->process_len += n;
            continue;
        }

        if (n < 0 && errno == EAGAIN) {
            break;
        }

        return false;
    }

    if (r->process_len == r->length) {
        r->process_len = 0;
        r->content_length = 0;
        r->phase = HttpPhase::RECV;

        return r->keep_alive;
    }

    return true;
}


bool RequestHandler::ParseRequestHeader(Request *r)
{
    // "GET /435fbb5e06ced5536952d7965a96088e/435fbb5e06ced5536952d7965a96088e HTTP/1.1"
    const int dir_start = sizeof("GET /") - 1;
    const int key_start = dir_start + KEY_CHAR_SIZE + 1;
    const int ver_start = key_start + KEY_CHAR_SIZE + sizeof(" HTTP/1.") - 1;

    r->method = r->buffer[0];
    r->version = r->buffer[ver_start];
    r->keep_alive = r->version == '1';

    if (strstr(r->buffer, "Connection: Keep-Alive")) {
        r->keep_alive = true;
    } else if (strstr(r->buffer, "Connection: Close")) {
        r->keep_alive = false;
    }

    r->dir.Load(r->buffer + dir_start);
    r->key.Load(r->buffer + key_start);

    r->http_code = 200;

    return true;
}

bool RequestHandler::GenrateResponseHeader(Request *r)
{
    const char* header_format = "HTTP/1.%c %d OK\r\nConnection: %s\r\nContent-Length: %d\r\n\r\n%s";
    char body[KEY_CHAR_SIZE * 2 + 3] = {'/', 0};

    r->dir.Dump(body+1);
    body[KEY_CHAR_SIZE + 1] = '/';
    r->key.Dump(body + KEY_CHAR_SIZE + 2);
    body[KEY_CHAR_SIZE  * 2 + 2] = '\0';

    int n = snprintf(r->buffer, BUF_SIZE, header_format, r->version, r->http_code, r->keep_alive ? "Keep-Alive" : "Close", KEY_CHAR_SIZE *2+2, body);

    if (n < 0) {
        return false;
    }

    r->length = n;

    return true;
}
