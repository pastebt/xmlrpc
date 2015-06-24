package xmlrpc

import (
	"os"
	"io"
	"fmt"
	"bytes"
	//"io/ioutil"
    "runtime"
	"reflect"
	"strings"
	"net/http"
	"encoding/xml"
)


type DFT []interface{}


type methodData struct {
	obj       interface{}
	//method    reflect.Method
    ftype       reflect.Type    // function/method type
    fvalue      reflect.Value   // function/method value
	padParams   bool
    dft         DFT
}

// Map from XML-RPC procedure names to Go methods
type Handler struct {
	methods map[string]*methodData
    logf    func(req *http.Request, code int, msg string)
}

// create a new handler mapping XML-RPC procedure names to Go methods
func NewHandler() *Handler {
	h := new(Handler)
	h.methods = make(map[string]*methodData)
	return h
}

func (h *Handler)SetLogf(logf func(*http.Request, int, string)) {
    h.logf = logf
}


// register all methods associated with the Go object, passing them
// through the name mapper if one is supplied
//
// The name mapper can return "" to ignore a method or transform the
// name as desired
func (h *Handler) Register(obj interface{}, mapper func(string) string,
	padParams bool) error {
	ot := reflect.TypeOf(obj)

	for i := 0; i < ot.NumMethod(); i++ {
		m := ot.Method(i)
		if m.PkgPath != "" {
			continue
		}

		var name string
		if mapper == nil {
			name = m.Name
		} else {
			name = mapper(m.Name)
			if name == "" {
				continue
			}
		}

		md := &methodData{obj: obj, ftype: m.Type, fvalue: m.Func, padParams: padParams}
		h.methods[name] = md
		h.methods[strings.ToLower(name)] = md
	}

	return nil
}


// register a func, if name is "", then use func name
func (h *Handler) RegFunc(f interface{}, name string, dft DFT) error {
	vo := reflect.ValueOf(f)
    if vo.Kind() != reflect.Func {
        panic("RegFunc only register function")
    }
    md := &methodData{obj: nil, ftype: vo.Type(), fvalue: vo, dft: dft}
    if name == "" {
        // runtime.FuncForPC always return pkg.func_name, so we cut prefix "main."
        name = runtime.FuncForPC(vo.Pointer()).Name()[5:]
    }
    h.methods[name] = md
    return nil
}


var faultType = reflect.TypeOf((*Fault)(nil))


// Return an XML-RPC fault
func writeFault(out io.Writer, code int, msg string) {
	fmt.Fprintf(out, `<?xml version="1.0"?>
<methodResponse>
  <fault>
	<value>
		<struct>
		  <member>
			<name>faultCode</name>
			<value><int>%d</int></value>
		  </member>
		  <member>
			<name>faultString</name>
			<value>`, code)
	err := xml.EscapeText(out, []byte(msg))
	fmt.Fprintf(out, `</value>
		  </member>
		</struct>
	</value>
  </fault>
</methodResponse>`)

	// XXX dump the error to Stderr for now
    if err != nil {
        fmt.Fprintf(os.Stderr, "Cannot write fault#%d(%s): %v\n",
                    code, msg, err)
    }
}


// semi-standard XML-RPC response codes
const (
	errNotWellFormed = -32700
	errUnknownMethod = -32601
	errInvalidParams = -32602
	errInternal      = -32603
)


func (mData *methodData)getVals(methodName string, args []interface{}, req *http.Request) (vals []reflect.Value, f *Fault) {

    // expecting arg number
    expArgs := mData.ftype.NumIn()
    // valus will be used to call function, +1 is for potential req
	vals = make([]reflect.Value, 0, expArgs + 1)
    x := 0

    if mData.obj != nil {
        // this function is a object's method, fill first val with obj
        vals = append(vals, reflect.ValueOf(mData.obj))
        x = x + 1
    }

    if expArgs > x && reflect.TypeOf(req) == mData.ftype.In(x) {
        // first request is *http.Request, we fill it
        vals = append(vals, reflect.ValueOf(req))
        x = x + 1
    }

    for _, arg := range args {
        vals = append(vals, reflect.ValueOf(arg))
    }


    ff := func() *Fault {
        f := Fault{errInvalidParams,
                   fmt.Sprintf("Bad number of parameters for method \"%s\","+
                               " (input %d != expect %d)",
                               methodName, len(args), expArgs - x)}
        return &f
    }

    // input and request match
    if len(vals) == expArgs { return }

    // can miss one or give more because IsVariadic is true
    if mData.ftype.IsVariadic() && len(vals) >= expArgs -1 { return }

    // input more
    if len(vals) > expArgs {
        f = ff()
        return
    }

    dl := len(mData.dft)

    for i := len(vals); i < expArgs; i++ {
        if dl > 0 && dl + i >= expArgs {
            vals = append(vals, reflect.ValueOf(mData.dft[dl - expArgs + i]))
        } else if mData.padParams {
            vals = append(vals, reflect.Zero(mData.ftype.In(i)))
        } else {
            f = ff()
            return
        }
    }
    return
}


// handle an XML-RPC request
func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

    methodName, params, err, fault := Unmarshal(req.Body)
    if err != nil {
        msg := fmt.Sprintf("Unmarshal error: %v", err)
        writeFault(resp, errNotWellFormed, msg)
        if h.logf != nil { h.logf(req, errNotWellFormed, msg) }
        return
    } else if fault != nil {
        writeFault(resp, fault.Code, fault.Msg)
        if h.logf != nil { h.logf(req, fault.Code, fault.Msg) }
        return
	}

    // try to get input arguments
    var args []interface{}
    var ok bool

    if args, ok = params.([]interface{}); !ok {
        args = make([]interface{}, 1, 1)
        args[0] = params
    }

    // try to find registered function by name
    var mData *methodData

    if mData, ok = h.methods[methodName]; !ok {
        msg := fmt.Sprintf("Unknown method \"%s\"", methodName)
        writeFault(resp, errUnknownMethod, msg)
        if h.logf != nil { h.logf(req, errUnknownMethod, msg) }
        return
    }

    // get values
    vals, f := mData.getVals(methodName, args, req)
    if f != nil {
        writeFault(resp, f.Code, f.Msg)
        if h.logf != nil { h.logf(req, f.Code, f.Msg) }
        return
    }

    if h.logf != nil {
        h.logf(req, 0, fmt.Sprintf("call method %v, input %v", methodName, vals))
    }
    // exec function
    rtnVals := mData.fvalue.Call(vals)

    if len(rtnVals) == 1 && reflect.TypeOf(rtnVals[0].Interface()) == faultType {
        if fault, ok := rtnVals[0].Interface().(*Fault); ok {
            writeFault(resp, fault.Code, fault.Msg)
            if h.logf != nil { h.logf(req, fault.Code, fault.Msg) }
            return
        }
    }

    mArray := make([]interface{}, len(rtnVals), len(rtnVals))
    for i := 0; i < len(rtnVals); i++ {
        mArray[i] = rtnVals[i].Interface()
    }

    buf := bytes.NewBufferString("")
    err = marshalArray(buf, "", mArray)
    if err != nil {
        msg := fmt.Sprintf("Failed to marshal %s: %v", methodName, err)
        writeFault(resp, errInternal, msg)
        if h.logf != nil { h.logf(req, errInternal, "ouput: " + msg) }
        return
    }
    //fmt.Fprintf(os.Stderr, buf.String())
    buf.WriteTo(resp)
}


/*
// handle an XML-RPC request
func (h *Handler) ServeHTTP_old(resp http.ResponseWriter, req *http.Request) {
  //b, _ := ioutil.ReadAll(req.Body)
  //body := string(b)
  //fmt.Fprintf(os.Stderr, body)
  //methodName, params, err, fault := UnmarshalString(body)
  //fmt.Fprintf(os.Stderr, "ServeHTTP params = %v\n", params)

	methodName, params, err, fault := Unmarshal(req.Body)

	if err != nil {
		writeFault(resp, errNotWellFormed,
			fmt.Sprintf("Unmarshal error: %v", err))
		return
	} else if fault != nil {
		writeFault(resp, fault.Code, fault.Msg)
		return
	}

	var args []interface{}
	var ok bool

	if args, ok = params.([]interface{}); !ok {
		args := make([]interface{}, 1, 1)
		args[0] = params
	}
  //fmt.Fprintf(os.Stderr, "%v", args)

	var mData *methodData

	if mData, ok = h.methods[methodName]; !ok {
		writeFault(resp, errUnknownMethod,
			fmt.Sprintf("Unknown method \"%s\"", methodName))
		return
	}

	expArgs := mData.ftype.NumIn()
    x := 0
    if mData.obj != nil { x = 1 }

    // flag to support func(...interface{})
    //y := mData.ftype.In(expArgs - 1) == istype
    // IsVariadic
    y := mData.ftype.IsVariadic()

    dl := 0
    if mData.dft != nil { dl = len(mData.dft) }

	if len(args) + x != expArgs && !y && len(args) + x + dl < expArgs {
		if !mData.padParams || len(args) + x > expArgs {
			writeFault(resp, errInvalidParams,
				fmt.Sprintf("Bad number of parameters for method \"%s\","+
					" (%d != %d)", methodName, len(args) + x, expArgs-1))
			return
		}
	}

	vals := make([]reflect.Value, expArgs, expArgs)

    i := x
    if x == 1 {
        vals[0] = reflect.ValueOf(mData.obj)
    }

	for ; i < expArgs; i++ {
		if (mData.padParams || (y && i == expArgs - 1) || dl > 0) && i >= len(args) {
            if dl > 0 && dl + i >= expArgs {
                vals[i] = reflect.ValueOf(mData.dft[dl - expArgs + i])
            } else {
                vals[i] = reflect.Zero(mData.ftype.In(i))
            }
			continue
		}

        if i == expArgs - 1 && y {
            vals[i] = reflect.ValueOf(args[i])
            for _, a := range args[i+1:] {
                vals = append(vals, reflect.ValueOf(a))
            }
            break
        }

		if !reflect.TypeOf(args[i-x]).ConvertibleTo(mData.ftype.In(i)) {
			writeFault(resp, errInvalidParams,
				fmt.Sprintf("Bad %s argument #%d (%v should be %v)",
					methodName, i-x, reflect.TypeOf(args[i-x]),
					mData.ftype.In(i)))
			return
		}

		vals[i] = reflect.ValueOf(args[i-x])
	}

	rtnVals := mData.fvalue.Call(vals)

	if len(rtnVals) == 1 && reflect.TypeOf(rtnVals[0].Interface()) == faultType {
		if fault, ok := rtnVals[0].Interface().(*Fault); ok {
			writeFault(resp, fault.Code, fault.Msg)
			return
		}
	}

	mArray := make([]interface{}, len(rtnVals), len(rtnVals))
	for i := 0; i < len(rtnVals); i++ {
		mArray[i] = rtnVals[i].Interface()
	}

	buf := bytes.NewBufferString("")
	err = marshalArray(buf, "", mArray)
	if err != nil {
		writeFault(resp, errInternal, fmt.Sprintf("Failed to marshal %s: %v",
			methodName, err))
		return
	}
  fmt.Fprintf(os.Stderr, buf.String())
	buf.WriteTo(resp)
}
*/

// start an XML-RPC server
/*
func StartServer(port int) *Handler {
	h := NewHandler()
	http.HandleFunc("/", h.HandleRequest)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	return h
}
*/
