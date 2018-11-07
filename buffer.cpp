#include "buffer.h"

const ssize_t FILE_RECV_TMP_SIZE = 4096;


MemBuffer::MemBuffer(size_t buf_size)
{
    size = buf_size;
    processed = recved = sended = 0;
    data_ = unique_ptr<char []>(new char[buf_size]);
}


void MemBuffer::Recv(int sock)
{
    while(recved < size) {
        ssize_t n = read(sock, data_.get() + recved, size - recved);
        if (n > 0) {
            recved += n;
        } else {
            if (n < 0 && errno == EAGAIN){
                return;
            }
            throw ReqError("MEM_RECV");
        }
    }
}


void MemBuffer::Send(int sock)
{
    while(sended < size) {
        ssize_t n = write(sock, data_.get() + sended, size - sended);
        if (n > 0) {
            sended += n;
        } else {
            if (n < 0 && errno == EAGAIN){
                return;
            }
            throw ReqError("MEM_SEND");
        }
    }
}


FileBuffer::FileBuffer(int fd, off_t off, ssize_t buf_size)
{
    fd_ = fd;
    off_ = off;
    size = buf_size;
    processed = recved = sended = 0;
}


void FileBuffer::Recv(int sock)
{
    if(!tmp_) {
        tmp_ = unique_ptr<char []>(new char[FILE_RECV_TMP_SIZE]);
    }

    while(recved < size) {
        bool again = false;
        ssize_t tmp_size = 0;
 
        while(recved < size && tmp_size < FILE_RECV_TMP_SIZE) {
            ssize_t remain = FILE_RECV_TMP_SIZE - tmp_size;
            if (remain < ssize_t(size - recved)) {
                remain = size - recved;
            }

            ssize_t n = read(sock, tmp_.get() + tmp_size, remain);
            if (n > 0) {
                recved += n;
                tmp_size += n;
            } else {
                if (n < 0 && errno == EAGAIN) {
                    again = true;
                    break;
                }
                throw ReqError("FILE_RECV");
            }
        }

        if (tmp_size > 0 && tmp_size != pwrite(fd_, tmp_.get(), tmp_size, off_ + recved)) {
            throw SvrError("FileBuffer::Recv pwrite failed", __FILE__, __LINE__);
        }

        if (again) {
            return;
        }
    }
}


void FileBuffer::Send(int sock)
{
    while (sended < size){
        off_t start = off_ + sended;
        ssize_t n = sendfile(sock, fd_, &start, size - sended);
        if (n > 0) {
            sended += n;
        } else if (n == 0 || errno != EAGAIN){
            throw ReqError("FILE_SEND");
        }
    }
}


void FileBuffer::Write(const char *buf, ssize_t size)
{
    if(pwrite(fd_, buf, size, off_ + recved) != size) {
        throw SvrError("FileBuffer::Write pwrite failed", __FILE__, __LINE__);
    }
    recved += size;
}
