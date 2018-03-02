package encode
//Structs must be constant size and any member must be exported (Capital first letter)

import (
	"encoding/binary"
	"bytes"
)

func Size(i interface{}) int{
	return binary.Size(i)
}

func  FromBytes(data []byte,i interface{}){
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, i)
	if err != nil {
		panic(err)
	}
}

func ToBytes(data interface{}) []byte{
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}