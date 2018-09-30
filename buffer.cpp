#include "buffer.h"

const int FILE_RECV_TMP_SIZE = 4096


MemBuffer::MemBuffer(size_t buf_size)
{
    size = buf_size;
    processed = recved = sended = 0;
    data = unique_ptr<char []>(new char[buf_size]);
}


bool MemBuffer::Recv(int sock)
{
    while(recved < size) {
        ssize_t n = read(sock, data_.get() + recved, size - recved);
        if (n > 0) {
            recved += n;
        } else {
            return n < 0 && errno == EAGAIN;
        }
    }
    return true;
}


bool MemBuffer::Send(int sock)
{
    while(sended < size) {
        ssize_t n = write(sock, data_.get() + sended, size - sended);
        if (n > 0) {
            sended += n;
        } else {
            return n < 0 && errno == EAGAIN;
        }
    }
    return true;
}


FileBuffer::FileBuffer(int fd, off_t off, size_t cap)
{
    fd_ = fd;
    off_ = off;
    size = buf_size;
    processed = recved = sended = 0;
}


bool FileBuffer::Recv(int sock)
{
    if(!tmp_) {
        tmp_ = unique_ptr<char []>(new char[FILE_RECV_TMP_SIZE])
    }

    while(recved < size)
        size_t tmp_size = 0;
 
        while(recved < size && tmp_size < FILE_RECV_TMP_SIZE) {
            size_t remain = FILE_RECV_TMP_SIZE - tmp_size;
            if (remain < size - recved) {
                remain = size - recved
            }

            ssize_t n = read(sock, data_.get() + tmp_size, remain);
            if (n < 0 && errno != EAGAIN) {
                return false;
            }

            recved += n;
            tmp_size += n;
        }

        if (tmp_size > 0 && tmp_size != pwrite(fd, tmp_.get(), buf_size, off_ + recved)) {
            return false;
        }

        if (tmp_size < FILE_RECV_TMP_SIZE) {
            break;
        }
    }

    return true;
}


bool FileBuffer::Send(int sock)
{
    while (sended < size){
        off_t start = off_ + sended;
        ssize_t n = sendfile(sock, fd_, &start, size);
        if (n > 0) {
            off_ += n;
        } else {
            return n < 0 && errno == EAGAIN;
        }
    }

    return true;
}
