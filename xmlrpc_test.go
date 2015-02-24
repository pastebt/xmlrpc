package xmlrpc

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func getTypeString(val interface{}, noSpaces bool) string {
	preSpace := "\n		"
	postSpace := "\n	  "

	var pre, post string
	if noSpaces {
		pre = ""
		post = ""
	} else {
		pre = preSpace
		post = postSpace
	}

	if val == nil {
		return pre + "<nil/>" + post
	}

	switch v := val.(type) {
	case bool:
		var bVal int
		if v {
			bVal = 1
		} else {
			bVal = 0
		}
		return fmt.Sprintf("%s<boolean>%d</boolean>%s", pre, bVal, post)
	case float64:
		// hack to make float values match
		fStr := fmt.Sprintf("%f", v)
		fLen := len(fStr)
		fSub := fStr[fLen-3 : fLen]
		if fLen > 3 && fSub != "000" {
			fStr += "000"
		}

		return fmt.Sprintf("%s<double>%s</double>%s", pre, fStr, post)
	case int:
		return fmt.Sprintf("%s<int>%d</int>%s", pre, v, post)
	case string:
		return v
	case (map[string]interface{}):
		valStr := fmt.Sprintf("%s<struct>", preSpace)
		for mkey, mval := range v {
			valStr += fmt.Sprintf(`
		  <member>
			<name>%s</name>
			<value>%v</value>
		  </member>`, mkey, getTypeString(mval, true))
		}
		valStr += fmt.Sprintf("%s</struct>%s", preSpace, postSpace)
		return valStr
	case time.Time:
		tag := "dateTime.iso8601"
		return fmt.Sprintf("%s<%s>%s</%s>%s", pre, tag,
			v.Format(ISO8601_LAYOUT), tag, post)
	}

	rval := reflect.ValueOf(val)
	if rval.Kind() == reflect.Array || rval.Kind() == reflect.Slice {
		buf := bytes.NewBufferString("\n		<array><data>\n")
		for i := 0; i < rval.Len(); i++ {
			buf.WriteString("<value>")
			wrapValue(buf, rval.Index(i))
			buf.WriteString("</value>\n")
		}
		buf.WriteString("</data></array>\n	  ")
		return buf.String()
	} else {
		fmt.Printf("Not handling Kind %v\n", rval.Kind())
	}

	return fmt.Sprintf("<???>%v(%T)</???>", val, val)
}

// Translate a local data object into an XML string
func marshalString(methodName string, args ...interface{}) (string, error) {
	buf := bytes.NewBufferString("")
	err := marshalArray(buf, methodName, args)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseAndCheck(t *testing.T, methodName string, expVal interface{},
	xmlStr string) {
	name, val, err, fault := UnmarshalString(xmlStr)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	} else if fault != nil {
		t.Fatalf("Returned fault %s", fault)
	}

	if name != methodName {
		if methodName == "" {
			t.Fatal("Did not expect method name \"%s\"", name)
		} else {
			t.Fatal("Expected method name \"%s\", not \"%s\"", methodName, name)
		}
	}

	if expVal == nil {
		if val != nil {
			t.Fatalf("Got unexpected value %v <%T>", val, val)
		}
	} else {
		if reflect.TypeOf(val) != reflect.TypeOf(expVal) {
			t.Fatalf("Returned type %T, not %T", val, expVal)
		}

		if !reflect.DeepEqual(val, expVal) {
			t.Fatalf("Returned value %v, not %v", val, expVal)
		}
	}
}

func parseUnimplemented(t *testing.T, methodName string, expVal interface{}) {
	xmlStr := wrapMethod(methodName, expVal)
	name, val, err, fault := UnmarshalString(xmlStr)
	if err == nil {
		t.Fatalf("Unimplemented type didn't return an error")
	} else if !strings.Contains(err.Error(), "nimplemented") {
		t.Fatalf("Returned unexpected error %s", err)
	}

	if fault != nil {
		t.Fatalf("Returned unexpected fault %s", fault)
	}

	if name != methodName {
		if methodName == "" {
			t.Fatal("Did not expect method name \"%s\"", name)
		} else {
			t.Fatal("Expected method name \"%s\", not \"%s\"", methodName, name)
		}
	}

	if val != nil {
		t.Fatalf("Got value %v from unimplemented type", val)
	}
}

func wrapAndParse(t *testing.T, methodName string, expVal interface{}) {
	xmlStr := wrapMethod(methodName, expVal)
	parseAndCheck(t, methodName, expVal, xmlStr)
}

func wrapMethod(methodName string, args ...interface{}) string {
	buf := bytes.NewBufferString("<?xml version=\"1.0\"?>\n")

	var backStr string
	if methodName == "" {
		fmt.Fprintf(buf, "<methodResponse>\n")
		backStr = "</methodResponse>"
	} else {
		fmt.Fprintf(buf, "<methodCall>\n  <methodName>%s</methodName>\n",
			methodName)
		backStr = "</methodCall>"
	}

	fmt.Fprintf(buf, "  <params>\n")
	for _, a := range args {
		fmt.Fprintf(buf, `	<param>
	  <value>%v</value>
	</param>
`, getTypeString(a, false))
	}
	fmt.Fprintf(buf, "  </params>\n%s\n", backStr)

	return string(buf.Bytes())
}

func TestMakeRequestBool(t *testing.T) {
	expVal := true
	methodName := "foo"

	xmlStr, err := marshalString(methodName, expVal)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := wrapMethod(methodName, expVal)
	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestMakeRequestDateTime(t *testing.T) {
	val := "19980717T14:08:55"
	expVal, err := time.Parse(ISO8601_LAYOUT, val)
	if err != nil {
		t.Errorf("Cannot create ISO8601 time \"%s\"\n", val)
	}

	methodName := "foo"

	xmlStr, err := marshalString(methodName, expVal)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := wrapMethod(methodName, expVal)
	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestMakeRequestInt(t *testing.T) {
	expVal := 123456
	methodName := "foo"

	xmlStr, err := marshalString(methodName, expVal)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := wrapMethod(methodName, expVal)
	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestMakeRequestArray(t *testing.T) {
	expVal := []int{1, 2, 3, 4}
	methodName := "foo"

	xmlStr, err := marshalString(methodName, expVal)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := wrapMethod(methodName, expVal)
	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestMakeRequestNil(t *testing.T) {
	var expVal interface{} = nil
	methodName := "foo"

	xmlStr, err := marshalString(methodName, expVal)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := wrapMethod(methodName, expVal)
	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestMakeRequestNoData(t *testing.T) {
	methodName := "foo"

	xmlStr, err := marshalString(methodName)
	if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	expStr := fmt.Sprintf(`<?xml version="1.0"?>
<methodCall>
  <methodName>%s</methodName>
  <params>
  </params>
</methodCall>
`, methodName)

	if xmlStr != expStr {
		t.Fatalf("Returned \"%s\", not \"%s\"", xmlStr, expStr)
	}
}

func TestParseRequestInt(t *testing.T) {
	wrapAndParse(t, "foo", 54321)
}

func XXXTestParseResponseArray(t *testing.T) {
	var array = []int{1, -1, 0, 1234567}
	wrapAndParse(t, "", array)
}

func TestParseResponseBase64(t *testing.T) {
	tnm := "base64"
	val := "eW91IGNhbid0IHJlYWQgdGhpcyE"
	parseUnimplemented(t, "", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
}

func TestParseResponseBool(t *testing.T) {
	const expVal = true

	xmlStr := wrapMethod("", expVal)

	parseAndCheck(t, "", expVal, xmlStr)
}

func TestParseResponseDatetime(t *testing.T) {
	val := "19980717T14:08:55"
	expVal, err := time.Parse(ISO8601_LAYOUT, val)
	if err != nil {
		t.Errorf("Cannot create ISO8601 time \"%s\"\n", val)
	}

	wrapAndParse(t, "", expVal)
}

func TestParseResponseDouble(t *testing.T) {
	wrapAndParse(t, "", 123.456)
}

func TestParseResponseFault(t *testing.T) {
	code := 1
	msg := "Some fault"
	xmlStr := fmt.Sprintf(`<?xml version="1.0"?>
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
			<value>%s</value>
		  </member>
		</struct>
	</value>
  </fault>
</methodResponse>`, code, msg)

	name, _, err, fault := UnmarshalString(xmlStr)
	if name != "" {
		t.Fatalf("Returned name %s", name)
	} else if err != nil {
		t.Fatalf("Returned error %s", err)
	}

	if fault == nil {
		t.Fatalf("No fault was returned")
	} else if fault.Code != code {
		t.Fatalf("Expected fault code %d, not %d", code, fault.Code)
	} else if fault.Msg != msg {
		t.Fatalf("Expected fault message %s, not %s", msg, fault.Msg)
	}
}

func TestParseResponseInt(t *testing.T) {
	wrapAndParse(t, "", 1279905716)
}

func TestParseResponseI4(t *testing.T) {
	tnm := "i4"
	val := -433221

	xmlStr := wrapMethod("", fmt.Sprintf("<%s>%v</%s>", tnm, val, tnm))
	parseAndCheck(t, "", val, xmlStr)
}

func TestParseResponseNil(t *testing.T) {
	wrapAndParse(t, "", nil)
}

func TestParseResponseNoData(t *testing.T) {
	xmlStr := `<?xml version="1.0"?>
<methodResponse>
  <params>
  </params>
</methodResponse>`

	parseAndCheck(t, "", nil, xmlStr)
}

func TestParseResponseString(t *testing.T) {
	wrapAndParse(t, "", "abc123")
}

func TestParseResponseStringEmpty(t *testing.T) {
	wrapAndParse(t, "", "")
}

func TestParseResponseStringRaw(t *testing.T) {
	const expVal = "abc123"

	xmlStr := fmt.Sprintf(`<?xml version='1.0'?>
<methodResponse>
  <params>
	<param>
	  <value>%s</value>
	</param>
  </params>
</methodResponse>`, expVal)

	parseAndCheck(t, "", expVal, xmlStr)
}

func TestParseResponseStringRawEmpty(t *testing.T) {
	xmlStr := `<?xml version='1.0'?>
<methodResponse>
  <params>
	<param>
	  <value></value>
	</param>
  </params>
</methodResponse>`

	parseAndCheck(t, "", "", xmlStr)
}

func TestParseResponseStringEscapedChars(t *testing.T) {
	xmlStr := `<?xml version='1.0'?>
<methodResponse>
  <params>
	<param>
	  <value>&lt;/value&gt;</value>
	</param>
  </params>
</methodResponse>`

	parseAndCheck(t, "", "</value>", xmlStr)
}

func TestParseResponseStruct(t *testing.T) {
	structMap := map[string]interface{}{
		"boolVal": true, "intVal": 18, "strVal": "foo",
	}
	wrapAndParse(t, "", structMap)
}
