package xmlrpc

import (
    "testing"
    "bytes"
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


func TestWriteFault(tst *testing.T) {
    w := bytes.NewBufferString("")
    writeFault(w, 123, "fault msg string")
}
