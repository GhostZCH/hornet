import requests


host = "http://127.0.0.1:1100/"



def basic_test():
    k = "1" * 32

    url = host + k
    requests.delete(url)

    rsp = requests.get(url)
    print(rsp.headers)
    assert rsp.status_code == 404

    rsp = requests.post(url, "0123456789")
    print rsp
    assert rsp.status_code == 201

    print "000000"

    rsp = requests.get(url)
    print "1111111"
    print rsp
    print rsp.headers
    assert rsp.status_code == 200
    assert rsp.headers["X-Via-Cache"] == "my-test-node hdd"
    assert rsp.content == "0123456789"

    # rsp = requests.get(url)
    # assert rsp.status_code == 200
    # assert rsp.headers["X-Via-Cache"] == "my-test-node ssd"

    # rsp = requests.get(url)
    # assert rsp.status_code == 200
    # assert rsp.headers["X-Via-Cache"] == "my-test-node mem"

    # requests.delete(url)


def delete_test():
    pass

basic_test()