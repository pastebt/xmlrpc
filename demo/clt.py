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
print(svr.SayHello2("12345<>&6"))
print(svr.RetStrs("AbCdEf"))
print(svr.RetIntStr("AbCdEf"))
print(svr.RetMapSI("AbCdEf"))
print("RetMapSIF: ", svr.RetMapSIF("AbCdEf"))
print(svr.RetMapSS("AbCdEf"))
print(svr.RetStruct("AbCdEf"))
print(svr.ttt("ttt AbCdEf"))
print(svr.mmm("mmm AbCdEf"))
print(svr.mmm("mmm AbCdEf", 2))
print(svr.mmm("mmm AbCdEf", 12, 3, 4))
print(svr.ddd("ddd AbCdEf", False))
print(svr.ddd("ddd AbCdEf"))
print(svr.rrr("ddd AbCdEf"))
print(svr.rrr("ddd AbCdEf", False))
