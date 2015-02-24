package main

import (
    "fmt"
    "net/http"
    gxr "xmlrpc"
)


type SO struct {
    name string
}

func (so *SO)SayHello(who string) string{
    return fmt.Sprintf("%s say Hello to %s", so.name, who)
}


func main() {
    h := gxr.NewHandler()
    h.Register(&SO{"MyName"}, nil, false)
    http.Handle("/rpc", h)
    http.ListenAndServe(":2345", nil)
}
