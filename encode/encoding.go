package encode
//Structs must be explicit size and any member must be public
//int not allowed (use int8, int16 so on)
//Arrays allowed, but not slices
//Pointers should not be used
//Nested structs allowed but these must also conform with the rules above

import (
	"encoding/binary"
	"bytes"
)

func Size(i interface{}) int{
	return binary.Size(i)
}

//i should be a pointer
func FromBytes(data []byte, i interface{}){
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