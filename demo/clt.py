#! /usr/bin/python

try:
    import xmlrpclib as xrc
except ImportError:
    import xmlrpc.client as xrc


#s = xrc.dumps(("str", 1, True, {"k1": 1, "k2": 2}), "testfunc")
s = xrc.dumps(("<str&#~!>", 1, True), "testfunc")
print(s)
print(xrc.dumps(({},), "testfunc"))
print(xrc.dumps(({"1": "ab", "2": "cd"},), "testfunc"))

svr = xrc.ServerProxy("http://127.0.0.1:2345/rpc")
print(svr.SayHello("12345<>&6"))
#print(svr.RetStrs("AbCdEf"))
#print(svr.RetIntStr("AbCdEf"))
#print(svr.RetMapIS("AbCdEf"))
print(svr.RetMapSS("AbCdEf"))
#print(svr.RetStruct("AbCdEf"))
print(svr.ttt("AbCdEf"))
