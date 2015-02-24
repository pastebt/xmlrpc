#! /usr/bin/python

try:
    import xmlrpclib as xrc
except ImportError:
    import xmlrpc.client as xrc


#s = xrc.dumps(("str", 1, True, {"k1": 1, "k2": 2}), "testfunc")
s = xrc.dumps(("str", 1, True), "testfunc")
print(s)

svr = xrc.ServerProxy("http://127.0.0.1:2345/rpc")
print(svr.SayHello("123456"))
