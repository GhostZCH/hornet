import requests

ADDR = ("127.0.0.1", 6969)

def test1():
    rsp = requests.get("http://127.0.0.1:6969/")
    print rsp
    pass
