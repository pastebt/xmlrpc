package xmlrpc

import (
	"encoding/xml"
	"fmt"
)

// internal XML parser tokens
const (
	// unknown token
	tokenUnknown = -3

	// marker for XML character data
	tokenText = -2

	// ignored XML data
	tokenProcInst = -1

	// keyword tokens
	tokenFault = iota
	tokenMember
	tokenMethodCall
	tokenMethodName
	tokenMethodResponse
	tokenName
	tokenParam
	tokenParams
	tokenValue

	// marker for data types
	tokenDataType

	// data type tokens
	tokenArray
	tokenBase64
	tokenBoolean
	tokenData
	tokenDateTime
	tokenDouble
	tokenInt
	tokenNil
	tokenString
	tokenStruct
)

// map token strings to constant values
var tokenMap map[string]int

// load the tokens into the token map
func initTokenMap() {
	tokenMap = make(map[string]int)
	tokenMap["array"] = tokenArray
	tokenMap["base64"] = tokenBase64
	tokenMap["boolean"] = tokenBoolean
	tokenMap["data"] = tokenData
	tokenMap["dateTime.iso8601"] = tokenDateTime
	tokenMap["double"] = tokenDouble
	tokenMap["fault"] = tokenFault
	tokenMap["int"] = tokenInt
	tokenMap["member"] = tokenMember
	tokenMap["methodCall"] = tokenMethodCall
	tokenMap["methodName"] = tokenMethodName
	tokenMap["methodResponse"] = tokenMethodResponse
	tokenMap["name"] = tokenName
	tokenMap["nil"] = tokenNil
	tokenMap["param"] = tokenParam
	tokenMap["params"] = tokenParams
	tokenMap["string"] = tokenString
	tokenMap["struct"] = tokenStruct
	tokenMap["value"] = tokenValue
}

type xmlToken struct {
	token   int
	isStart bool
	text    string
}

func (tok *xmlToken) Is(val int) bool {
	return tok.token == val
}

func (tok *xmlToken) IsDataType() bool {
	return tok.token > tokenDataType
}

func (tok *xmlToken) IsNone() bool {
	return tok.token == tokenProcInst
}

func (tok *xmlToken) IsStart() bool {
	return tok.token >= 0 && tok.isStart
}

func (tok *xmlToken) IsText() bool {
	return tok.token == tokenText
}

func (tok *xmlToken) Name() string {
	return getTokenName(tok.token)
}

func getTokenName(token int) string {
	if token == tokenProcInst {
		return "ProcInst"
	}

	for k, v := range tokenMap {
		if v == token {
			return k
		}
	}

	return fmt.Sprintf("??#%d??", token)
}

func (tok *xmlToken) Text() string {
	if tok.token != tokenText {
		return ""
	}

	return tok.text
}

func (tok *xmlToken) String() string {
	if tok.token == tokenText {
		return fmt.Sprintf("\"%v\"", tok.text)
	}

	var slash string
	if tok.isStart {
		slash = ""
	} else {
		slash = "/"
	}

	return fmt.Sprintf("{%s%s#%d}", slash, tok.Name(), tok.token)
}

func getTagToken(tag string) (int, error) {
	if tok, ok := tokenMap[tag]; ok {
		return tok, nil
	} else if tag == "i4" {
		return tokenInt, nil
	} else {
		return tokenUnknown, fmt.Errorf("Unknown tag <%s>", tag)
	}
}

func getNextToken(p *xml.Decoder) (*xmlToken, error) {
	tag, err := p.Token()
	if tag == nil || err != nil {
		return nil, err
	}

	if tokenMap == nil {
		initTokenMap()
	}

	switch v := tag.(type) {
	case xml.StartElement:
		tok, err := getTagToken(v.Name.Local)
		if err != nil {
			return nil, err
		}

		return &xmlToken{token: tok, isStart: true}, nil
	case xml.EndElement:
		tok, err := getTagToken(v.Name.Local)
		if err != nil {
			return nil, err
		}

		return &xmlToken{token: tok, isStart: false}, nil
	case xml.CharData:
		return &xmlToken{token: tokenText, text: string(v)}, nil
	case xml.ProcInst:
		return &xmlToken{token: tokenProcInst}, nil
	default:
		return nil, fmt.Errorf("Not handling XML token %v (type %T)", v, v)
	}
}
