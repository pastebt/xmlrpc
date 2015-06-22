#xmlrpc

Most code and doc fork from https://code.google.com/p/go-xmlrpc/

Package xmlrpc provides a rudimentary interface for sending and receiving
XML-RPC requests.

Procedures are provided by any objects registered with the server.
An XML-RPC server is:

```go
    type SomeObject struct {
        size int
    }

    func (so *SomeObject) GetSize() int { return so.size }
    func (so *SomeObject) SetSize(size int) { so.size = size }
    h := xmlrpc.NewHandler()
    h.Register(&SomeObject{}, nil, false)

    http.Handle("/rpc", h)
    http.ListenAndServe(":2345", nil)
```
This will add 'GetSize' and 'SetSize' to the server.

The second parameter of the Register method is a name mapping function.  This
mapping function takes a method name as a parameter and can return "" to
ignore a method or return a transformed string.

As an example, this function will only accept methods starting with 'RPC'.
It will replace the initial "RPC" with "xmlrpc." and convert the first
character after 'RPC' to lowercase, so "GetSize" would be ignored and
"RPCGetSize" would be registered as "xmlrpc.getSize":
```go
    func clientMapper(methodname string) string {
        if !strings.HasPrefix(methodname, "RPC") {
            return ""
        }

        var buf bytes.Buffer
        
        buf.WriteString("xmlrpc.")
        r, n := utf8.DecodeRuneInString(methodname[3:])
        buf.WriteRune(unicode.ToLower(r))
        buf.WriteString(methodname[3+n:])

        return buf.String()
    }
```

Clients are created with xmlrpc.NewClient(host string, int port):

```go
    client, err := xmlrpc.NewClient("localhost", 2345)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Cannot create XML-RPC client: %v\n", err)
        return
    }
```
Remote procedure calls are made using client.RPCCall, whose parameters are
the name of the remote procedure along with any needed parameters:
```go
    reply, cerr, fault := client.RPCCall("SetThing", 123, "abc")
    if cerr != nil {
        fmt.Fprintf(os.Stderr, "Cannot call SetThing: %v\n", cerr)
        return
    } else if fault != nil {
        fmt.Fprintf(os.Stderr, "Exception from SetThing: %v\n", fault)
        return
    }

    fmt.Printf("SetThing(123, \"abc\") returned %v\n", reply)
```
(Note that parameters are optional so client.RPCCall("foo") is valid code.)


You also can register a function using RegFunc(f interface, name string, padParams bool)
Which will register f, with name, if name is "", thhem use name of f
