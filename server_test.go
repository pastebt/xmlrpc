package xmlrpc

import (
    "testing"
    "bytes"
    "net/http"
    "net/http/httptest"
)


func a () bool {
    return true
}


func TestServer(tst *testing.T) {
    h := NewHandler()
    bf := func (p string) string { return p }
    cf := func (p, q string) int { return len(p + q) }
    h.RegFunc(a, "", nil)
    h.RegFunc(bf, "B", nil)
    h.RegFunc(cf, "", nil)
    tst.Logf("method list = %v", h.GetMethodList())
}


func TestSetLog(tst *testing.T) {
    h := NewHandler()
    h.SetLogf(func(r *http.Request, l int, n string) {})
}


type A struct {
    http.Request
    i int
}
//func (a *A)Add(b int) int { return int(*a) + b }
func (a *A)Add(b int) int { return a.i + b }
func (a *A)Del(b int) int { return a.i - b }

func TestRegister(tst *testing.T) {
    h := NewHandler()
    h.Register(func (){}, func(n string)string{return n}, true)
    h.Register(new(A), func(n string)string{if n == "Del" {return ""}; return n}, true)
    h.Register(new(A), nil, true)
}


func TestWriteFault(tst *testing.T) {
    w := bytes.NewBufferString("")
    writeFault(w, 123, "fault msg string")
}


func TestServeHTTP(tst *testing.T) {
    h := NewHandler()
    h.SetLogf(func(r *http.Request, l int, n string) {})
    // bad xml format
    buf := bytes.NewBufferString("")
    buf.Write([]byte(`<?xml version="1.0"?><ethodResponse`))
    req, err := http.NewRequest("GET", "/rpc", buf)
    if err != nil {
        tst.Error(err)
    }
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    b := w.Body.String()
    tst.Logf("code=%d, body=%s", w.Code, b)

    buf = bytes.NewBufferString("")
    //err := xmlrpc.Marshal(buf, "update458", "rule458.txt")
    err = Marshal(buf, "funcName", "data")
    if err != nil {
        tst.Error(err)
    }
    req, err = http.NewRequest("GET", "/rpc", buf)
    if err != nil {
        tst.Error(err)
    }
    w = httptest.NewRecorder()

    h.ServeHTTP(w, req)

    b = w.Body.String()
    tst.Logf("code=%d, body=%s", w.Code, b)

}
