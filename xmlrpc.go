package xmlrpc

import (
    "io"
//    "os"  
    "fmt"
    "time"
    "bytes"
    "errors"
    "reflect"
    "strconv"
    "strings"
    "net/url"
    "net/http"
    "encoding/xml"
)


// python list, a list can have different type item
type LIST []interface{}

// python dict, a map can have different type value
type DICT map[string]interface{}


// A Fault represents an error or exception in the procedure call
// being run on the remote machine
type Fault struct {
	Code int
	Msg  string
}

func NewFault(code int, msg string) *Fault {
	return &Fault{Code: code, Msg: msg}
}

// Return a string representation of the XML-RPC fault
func (f *Fault) String() string {
	if f == nil {
		return "NilFault"
	}
	return fmt.Sprintf("%s (code#%d)", f.Msg, f.Code)
}

func extractParams(v []interface{}) interface{} {
	if len(v) == 0 {
		return nil
    }
	if len(v) == 1 {
		return v[0]
	}
	return v
}

// get the method name from the <methodResponse>
func getMethodName(p *xml.Decoder) (string, error) {
	var methodName string

	inName := false
	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return "", errors.New("Unexpected end-of-file in getMethodName()")
		} else if err != nil {
			return "", err
		}

		if tok.IsText() {
			if !inName {
				// ignore text outside <methodName> and </methodName>
			} else {
				if methodName != "" {
					return "", fmt.Errorf("Multiple method names"+
						" (\"%s\" and \"%s\")", methodName, tok.Text())
				}

				methodName = tok.Text()
			}

			continue
		}

		if tok.Is(tokenMethodName) {
			if !tok.IsStart() {
				if !inName {
					return "", errors.New("Got </methodName> without" +
						" <methodName>")
				}

				break
			}

			inName = tok.IsStart()

			continue
		}

		return "", fmt.Errorf("Unexpected methodName token %s", tok)
	}

	return methodName, nil
}

// extract the method data
func getMethodData(p *xml.Decoder) ([]interface{}, *Fault, error) {
	var params = make([]interface{}, 0)
	var fault *Fault

	// state variables
	inParams := false
	inParam := false
	inFault := false

	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return nil, nil, errors.New("Unexpected end-of-file in" +
				" getMethodData()")
		} else if err != nil {
			return nil, nil, err
		}

		if tok.Is(tokenParams) {
			if !tok.IsStart() {
				// found end marker for tag, so we're done
				break
			}

			inParams = true
			continue
		} else if inParams {
			if tok.Is(tokenParam) {
				inParam = tok.IsStart()
				continue
			} else if inParam {
				p, perr := getValue(p)
				if perr != nil {
					return nil, nil, perr
				}

				params = append(params, p)
				inParam = false
			}
		}

		if tok.Is(tokenFault) {
			if !tok.IsStart() {
				// found end marker for tag, so we're done
				break
			}

			inFault = true
			continue
		} else if inFault {
			var ferr error
			fault, ferr = getFault(p)
			if ferr != nil {
				return nil, nil, ferr
			}

			inFault = false
		}

		if !tok.IsText() {
			err = fmt.Errorf("Unexpected methodData token %s", tok)
			return nil, nil, err
		}
	}

	return params, fault, nil
}

// get the XML-RPC fault
func getFault(p *xml.Decoder) (*Fault, error) {
	val, err := getValue(p)
	if err != nil {
		return nil, err
	}

	fmap := val.(map[string]interface{})

	return &Fault{Code: fmap["faultCode"].(int),
		Msg: fmap["faultString"].(string)}, nil
}

// parse a <value>
func getValue(p *xml.Decoder) (interface{}, error) {
	var value interface{}

	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return nil, errors.New("Unexpected end-of-file in getValue()")
		} else if err != nil {
			return nil, err
		}

		if tok.Is(tokenValue) {
			if !tok.IsStart() {
				// found end marker for tag, so we're done
				break
			}

			var sawEndValue bool
			value, sawEndValue, err = getValueData(p)
			if err != nil {
				return nil, err
			} else if sawEndValue {
				if value == nil {
					value = ""
				}

				break
			}

			continue
		}

		if !tok.IsText() {
			err = fmt.Errorf("Unexpected value token %v", tok)
			return nil, err
		}
	}

	return value, nil
}

// parse the <value> data
func getValueData(p *xml.Decoder) (interface{}, bool, error) {
	var toktype = tokenUnknown
	var value interface{}
	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return nil, false, errors.New("Unexpected end-of-file" +
				" in getValue()")
		} else if err != nil {
			return nil, false, err
		}

		if tok.IsDataType() {
			if tok.IsStart() {
				if toktype == tokenUnknown {
					toktype = tok.token
					value, err = getData(p, tok)
					if err != nil {
						return nil, false, err
					}
				} else {
					msg := "Found multiple starting tokens in getValueData()"
					return nil, false, errors.New(msg)
				}
			} else {
				if !tok.Is(toktype) {
					err = fmt.Errorf("Unexpected valueData token %s", tok)
					return nil, false, err
				}

				// found end marker for tag, so we're done
				break
			}
		} else if tok.IsText() {
			if value == nil {
				value = tok.Text()
			}
		} else if tok.Is(tokenValue) {
			return value, true, nil
		} else {
			err = fmt.Errorf("Unexpected valueData token %s", tok)
			return nil, false, err
		}
	}

	return value, false, nil
}

// parse a <struct>
func getStruct(p *xml.Decoder) (map[string]interface{}, error) {
	var data = make(map[string]interface{})

	// state variables
	inStruct := true
	inMember := false
	inName := false

	var name string
	gotName := false

	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return nil, errors.New("Unexpected end-of-file in getStruct()")
		} else if err != nil {
			return nil, err
		}

		if tok.Is(tokenStruct) {
			if !tok.IsStart() {
				// found end marker for tag, so we're done
				break
			}

			inStruct = true
			continue
		} else if inStruct {
			if tok.Is(tokenMember) {
				inMember = tok.IsStart()
				gotName = false
				continue
			} else if inMember {
				if tok.Is(tokenName) {
					inName = tok.IsStart()
					if !inName {
						gotName = true
					}

					if gotName && !inName {
						value, verr := getValue(p)
						if verr != nil {
							return nil, verr
						}

						data[name] = value
						gotName = false
					}

					continue
				} else if inName && tok.IsText() {
					name = tok.Text()
				}
			}
		}

		if !tok.IsText() {
			err = fmt.Errorf("Unexpected struct token %s", tok)
			return nil, err
		}
	}

	return data, nil
}

// parse an <array>
func getArray(p *xml.Decoder) (interface{}, error) {
	var data = make([]interface{}, 0)

	// state variables
	inArray := true
	inData := false

	for {
		tok, err := getNextToken(p)
		if tok == nil {
			return nil, errors.New("Unexpected end-of-file in getArray()")
		} else if err != nil {
			return nil, err
		}

		if tok.Is(tokenArray) {
			if !tok.IsStart() {
				// found end marker for tag, so we're done
				break
			}

			inArray = true
			continue
		} else if inArray {
			if tok.Is(tokenData) {
				inData = tok.IsStart()
				continue
			} else if inData {
				if tok.Is(tokenValue) {
					if tok.IsStart() {
						value, sawEndValue, verr := getValueData(p)
						if verr != nil {
							return nil, verr
						} else if sawEndValue {
							if value == nil {
								value = ""
							}
						}

						data = append(data, value)
					}
				}

				continue
			}
		}

		if !tok.IsText() {
			err = fmt.Errorf("Unexpected array token %s", tok)
			return nil, err
		}
	}

/*
	if data == nil {
		return nil, nil
	}

	var array = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(data[0])),
		len(data), len(data))
	for i := 0; i < len(data); i++ {
		v := reflect.ValueOf(data[i])
fmt.Printf("#%d append %v<%T> to %v<%T>\n", i, v, v, array, array)
		//array = appendValue(array, data[i])
		array = reflect.Append(array, v)
	}

	return array.Slice(0, array.Len()), nil
*/

	return data, nil
}

// parse either a raw string or a <string>xxx</string>
func getText(p *xml.Decoder) (string, error) {
	tok, err := getNextToken(p)
	if tok == nil {
		return "", errors.New("Unexpected end-of-file in getText()")
	} else if err != nil {
		return "", err
	}

	if tok.Is(tokenString) && !tok.IsStart() {
		return "", nil
	} else if !tok.IsText() {
		return "", fmt.Errorf("Unexpected token %s in getText()", tok)
	}

	return tok.Text(), nil
}

const ISO8601_LAYOUT = "20060102T15:04:05"

func getDateISO8601(p *xml.Decoder) (interface{}, error) {
	valStr, err := getText(p)
	if err != nil {
		return nil, err
	}

	return time.Parse(ISO8601_LAYOUT, valStr)
}

// convert the XML-RPC to Go data
func getData(p *xml.Decoder, tok *xmlToken) (interface{}, error) {
	var valStr string
	var err error

	switch tok.token {
	case tokenArray:
		return getArray(p)
	case tokenBase64:
		return nil, errors.New("parseDataString(base64) unimplemented")
	case tokenBoolean:
		valStr, err = getText(p)
		if err != nil {
			return nil, err
		}

		if valStr == "1" {
			return true, nil
		} else if valStr == "0" {
			return false, nil
		} else {
			return nil, fmt.Errorf("Bad <boolean> value \"%s\"", valStr)
		}
	case tokenDateTime:
		return getDateISO8601(p)
	case tokenDouble:
		valStr, err = getText(p)
		if err != nil {
			return nil, err
		}

		f, ferr := strconv.ParseFloat(valStr, 64)
		if ferr != nil {
			return nil, ferr
		}

		return f, nil
	case tokenInt:
		valStr, err = getText(p)
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(valStr)
		//i, err := strconv.ParseInt(valStr, 10, 0)     // not sure which is right
		if err != nil {
			return nil, err
		}

		return i, nil
	case tokenNil:
		return nil, nil
	case tokenString:
		valStr, err = getText(p)
        //fmt.Fprintf(os.Stderr, "valStr = [%v]", valStr)
		if err != nil {
			return nil, err
		}

		return valStr, nil
	case tokenStruct:
		return getStruct(p)
	default:
		break
	}

	return nil, fmt.Errorf("Unknown type %s oin getData()", tok.Name())
}

// Translate an XML stream into a local data object
func Unmarshal(r io.Reader) (string, interface{}, error, *Fault) {
    if r == nil {
        return "", nil, fmt.Errorf("reader is nil"), nil
    }
	p := xml.NewDecoder(r)

	var methodName string
	var params []interface{}
	var fault *Fault

	isResp := false
	for {
		tok, err := getNextToken(p)
		if tok == nil {
			break
		} else if err != nil {
			return "", nil, err, nil
		}

		if tok.IsNone() || tok.IsText() {
			continue
		}

		if tok.Is(tokenMethodResponse) {
			if !tok.IsStart() {
				break
			}

			isResp = tok.IsStart()
		} else if tok.Is(tokenMethodCall) {
			if !tok.IsStart() {
				break
			}
		} else {
			err := fmt.Errorf("Unrecognized tag <%s>", tok.Name())
			return "", nil, err, nil
		}

		if !isResp && tok.IsStart() {
			var merr error
			methodName, merr = getMethodName(p)
			if merr != nil {
				return "", nil, merr, nil
			}
		}

		var perr error
		params, fault, perr = getMethodData(p)
		if perr != nil {
			return "", nil, perr, nil
		}
	}
  //fmt.Fprintf(os.Stderr, "params = %v\n", params)
	//return methodName, extractParams(params), nil, fault
	return methodName, params, nil, fault
}

// Translate an XML string into a local data object
func UnmarshalString(s string) (string, interface{}, error, *Fault) {
	return Unmarshal(strings.NewReader(s))
}

// translate an array into XML
func wrapArray(w io.Writer, val reflect.Value) error {
	fmt.Fprintf(w, "<array><data>\n")

	for i := 0; i < val.Len(); i++ {
		fmt.Fprintf(w, "<value>")
		aerr := wrapValue(w, val.Index(i))
		if aerr != nil {
			return aerr
		}
		fmt.Fprintf(w, "</value>\n")
	}

	fmt.Fprintf(w, "</data></array>")
	return nil
}

// translate an map[string]interface{} into XML
func wrapMap(w io.Writer, val reflect.Value) error {
    ks := val.MapKeys()
    if len(ks) < 1 {
        //return fmt.Errorf("Empty Map")
        fmt.Fprintf(w, "<struct>\n</struct>")
        return nil
    }
    if ks[0].Kind() != reflect.String {
        return fmt.Errorf("Only support map[string]interface{}, got %v", val)
    }
    fmt.Fprintf(w, "<struct>\n")
    for _, k := range ks {
        fmt.Fprintf(w, "<member>\n")
        fmt.Fprintf(w, "<name>%s</name>\n", k)
        fmt.Fprintf(w, "<value>")
        ret := wrapValue(w, val.MapIndex(k))
        if ret != nil { return ret }
        fmt.Fprintf(w, "</value>\n</member>\n")
    }
    fmt.Fprintf(w, "</struct>")
    return nil
    //return fmt.Errorf("Not wrapping type %v (%v)", val.Kind().String(), val)
}


// translate an map[string]interface{} into XML
func wrapStruct(w io.Writer, val reflect.Value) error {
    fmt.Fprintf(w, "<struct>\n")
    for i := 0; i < val.NumField(); i++ {
        f := val.Type().Field(i)
        fmt.Fprintf(w, "<member>\n")
        fmt.Fprintf(w, "<name>%s</name>\n", f.Name)
        fmt.Fprintf(w, "<value>")
        ret := wrapValue(w, val.Field(i))
        if ret != nil { return ret }
        fmt.Fprintf(w, "</value>\n</member>\n")
    }
    fmt.Fprintf(w, "</struct>")
    return nil
}


// translate a parameter into XML
func wrapParam(w io.Writer, i int, xval interface{}) error {
	var valStr string

	fmt.Fprintf(w, "	<param>\n	  <value>\n		")
	if xval == nil {
		valStr = "<nil/>"
	} else {
		err := wrapValue(w, reflect.ValueOf(xval))
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "%s\n	  </value>\n	</param>\n", valStr)

	return nil
}

// cached time.Time reflect.Type value
var timeType reflect.Type

// translate Go data into XML
func wrapValue(w io.Writer, val reflect.Value) error {
	var isError = false

	switch val.Kind() {
	case reflect.Bool:
		 bval := 0
		if val.Bool() {
			bval = 1
		}
		fmt.Fprintf(w, "<boolean>%d</boolean>", bval)
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(w, "<double>%f</double>", val.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(w, "<int>%d</int>", val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(w, "<int>%d</int>", val.Uint())
	case reflect.String:
        //fmt.Fprintf(w, "<string>%s</string>", val.String())
        fmt.Fprintf(w, "<string>")
        err := xml.EscapeText(w, []byte(val.String()))
        if err != nil {
            return fmt.Errorf("Failed to wrapping type %v (%v), err=%s",
                              val.Kind().String(), val, err.Error())
        }
        fmt.Fprintf(w, "</string>")
	//case reflect.Uint:
	//	isError = true
	//case reflect.Uint8:
	//	isError = true
	//case reflect.Uint16:
	//	isError = true
	//case reflect.Uint32:
	//	isError = true
	//case reflect.Uint64:
	//	isError = true
	case reflect.Uintptr:
		isError = true
	case reflect.Complex64:
		isError = true
	case reflect.Complex128:
		isError = true
	case reflect.Array:
		return wrapArray(w, val)
	case reflect.Chan:
		isError = true
	case reflect.Func:
		isError = true
	case reflect.Interface:
		//isError = true
        return wrapValue(w, val.Elem())
	case reflect.Map:
		//isError = true
		return wrapMap(w, val)
	case reflect.Ptr:
		isError = true
	case reflect.Slice:
		return wrapArray(w, val)
	case reflect.Struct:
		if timeType == nil {
			timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
		}

		if !val.Type().ConvertibleTo(timeType) {
			//isError = true
            return wrapStruct(w, val)
		} else {
			method := val.MethodByName("Format")
			params := []reflect.Value{reflect.ValueOf(ISO8601_LAYOUT)}
			rtn := method.Call(params)

			tag := "dateTime.iso8601"
			fmt.Fprintf(w, "<%s>%s</%s>", tag, rtn[0].String(), tag)
		}
	case reflect.UnsafePointer:
		isError = true
	default:
		return fmt.Errorf("Unknown Kind %v for %T (%v)", val.Kind(), val, val)
	}

	if isError {
		return fmt.Errorf("Not wrapping type %v (%v)", val.Kind().String(), val)
	}

	return nil
}

// Write a local data object as an XML-RPC request
func Marshal(w io.Writer, methodName string, args ...interface{}) error {
	return marshalArray(w, methodName, args)
}

// Write an array of zero or more data objects as an XML-RPC request
func marshalArray(w io.Writer, methodName string, args []interface{}) error {
	var name string
	var addExtra bool
	if methodName == "" {
		name = "Response"
		addExtra = false
	} else {
		name = "Call"
		addExtra = true
	}

	fmt.Fprintf(w, "<?xml version=\"1.0\"?>\n<method%s>\n", name)
	if addExtra {
		fmt.Fprintf(w, "  <methodName>%s</methodName>\n", methodName)
	}

	fmt.Fprintf(w, "  <params>\n")

	for i, a := range args {
		err := wrapParam(w, i, a)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(w, "  </params>\n</method%s>\n", name)

	return nil
}

// XML-RPC client data
type Client struct {
	http.Client
	urlStr string
}

// connect to a remote XML-RPC server
func NewClient(host string, port int) (*Client, error) {
	address := fmt.Sprintf("http://%s:%d/RPC2", host, port)

	uurl, uerr := url.Parse(address)
	if uerr != nil {
		return nil, uerr
	}

	return &Client{urlStr: uurl.String()}, nil
}

// call a procedure on a remote XML-RPC server
func (c *Client) RPCCall(methodName string,
	args ...interface{}) (interface{}, error, *Fault) {

	buf := bytes.NewBufferString("")
	berr := marshalArray(buf, methodName, args)
	if berr != nil {
		return nil, berr, nil
	}

	req, err := http.NewRequest("POST", c.urlStr,
		strings.NewReader(buf.String()))
	if err != nil {
		return nil, err, nil
	}

	req.Header.Add("Content-Type", "text/xml")

	r, err := c.Do(req)
	if err != nil {
		return nil, err, nil
	} else if r == nil {
		err = fmt.Errorf("PostString for %s returned nil response\n",
			methodName)
		return nil, err, nil
	}

	_, pval, perr, pfault := Unmarshal(r.Body)

	if r.Close {
		r.Body.Close()
	}

	return pval, perr, pfault
}
