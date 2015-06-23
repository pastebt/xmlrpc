package main

import (
    "fmt"
    "strings"
    "net/http"
    gxr "xmlrpc"
)


type SO struct {
    name string
}

func (so *SO)SayHello(who string) string{
    return fmt.Sprintf("%s say <>& Hello to %s", so.name, who)
}

func (so *SO)RetStrs(who string) (a, b string){
    a = fmt.Sprintf("%s return lower string %s", so.name, strings.ToLower(who))
    b = fmt.Sprintf("%s return upper string %s", so.name, strings.ToUpper(who))
    return
}

func (so *SO)RetIntStr(who string) (i int, s string) {
    i = int(who[0])
    s = strings.ToUpper(who)
    return
}

func (so *SO)RetMapIS(who string) (ret map[int]string) {
    ret = make(map[int]string)
    ret[int(who[0])] = who[2:]
    ret[int(who[1])] = who[3:]
    return
}

func (so *SO)RetMapSS(who string) (ret map[string]string) {
    ret = make(map[string]string)
    ret[who[:1]] = who[2:]
    ret[who[:2]] = who[3:]
    return
}

type TST struct {
    Name string
    Addr string
}
func (so *SO)RetStruct(who string) (ret TST) {
    ret = TST{}
    ret.Name = who
    ret.Addr = "address of " + who
    return ret
}

func ttt(in string) string {
    return "haha " + in
}

func mmm(in string, cc ...interface{}) string {
    return "haha " + in + "cc: " + fmt.Sprintf("%v", cc)
}

func ddd(in string, cc bool) string {
    return fmt.Sprintf("in=%v, cc=%v", in, cc)
}

func main() {
    h := gxr.NewHandler()
    h.Register(&SO{"MyName"}, nil, false)
    h.RegFunc(ttt, "", nil)
    h.RegFunc(mmm, "", nil)
    h.RegFunc(ddd, "", gxr.DFT{true})
    http.Handle("/rpc", h)
    panic(http.ListenAndServe(":2345", nil))
}
