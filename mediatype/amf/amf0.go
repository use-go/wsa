package amf

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/use-go/websocket-streamserver/logger"
)

//Data Type of AMF Data
const (
	TAMF0Number        = 0x00
	TAMF0Boolean       = 0x01
	TAMF0String        = 0x02
	TAMF0Object        = 0x03 //hash table=amfstring:key ,amftype:value,0x 00 00 09 end
	TAMF0MovieClip     = 0x04 //Not available in Remoting
	TAMF0Null          = 0x05
	TAMF0Undefined     = 0x06
	TAMF0Reference     = 0x07
	TAMF0EcmaArray     = 0x08 //object with size of hashTable
	TAMF0EndOfObject   = 0x09 //See Object ，表示object结束
	TAMF0StrictArray   = 0x0a //arrycount+propArray
	TAMF0Date          = 0x0b
	TAMF0LongString    = 0x0c
	TAMF0Unsupported   = 0x0d
	TAMF0RecordSet     = 0x0e //Remoting, server-to-client only
	TAMF0xmlDocument   = 0x0f
	TAMF0TypedObject   = 0x10
	TAMF0AvmplusObject = 0x11 //AMF3 data Sent by Flash player 9+
)

type AMF0Object struct {
	Props list.List
}
type AMF0Property struct {
	PropType int32
	Name     string
	Value    AMF0Data
}
type AMF0Data struct {
	StrValue  string
	NumValue  float64
	S16Value  int16
	BoolValue bool
	ObjValue  AMF0Object
}

type AMF0Encoder struct {
	writer *bytes.Buffer
}

func (amf0Encoder *AMF0Encoder) Init() {
	amf0Encoder.writer = new(bytes.Buffer)
}

func (amf0Encoder *AMF0Encoder) EncodeString(str string) (err error) {
	length := uint32(len(str))
	if length == 0 {
		return errors.New("invald string")
	}
	if length >= 0xffff {
		err = amf0Encoder.writer.WriteByte(TAMF0LongString)
		if err != nil {
			return err
		}
		err = binary.Write(amf0Encoder.writer, binary.BigEndian, &length)
		if err != nil {
			return err
		}
	} else {
		err = amf0Encoder.writer.WriteByte(TAMF0String)
		if err != nil {
			return err
		}
		var tmp16 uint16
		tmp16 = uint16(length)
		err = binary.Write(amf0Encoder.writer, binary.BigEndian, &tmp16)
		if err != nil {
			return err
		}
	}

	_, err = amf0Encoder.writer.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeNumber(num float64) (err error) {
	err = amf0Encoder.writer.WriteByte(TAMF0Number)
	if err != nil {
		return err
	}
	err = binary.Write(amf0Encoder.writer, binary.BigEndian, &num)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeNU() {

}

func (amf0Encoder *AMF0Encoder) EncodeBool(boo bool) (err error) {
	err = amf0Encoder.writer.WriteByte(TAMF0Boolean)
	if err != nil {
		return err
	}
	var tmp8 byte
	if boo {
		tmp8 = 1
	} else {
		tmp8 = 0
	}
	err = amf0Encoder.writer.WriteByte(tmp8)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeInt16(num int16) (err error) {
	err = binary.Write(amf0Encoder.writer, binary.BigEndian, &num)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeInt24(num int32) (err error) {
	var tmp8 int8
	tmp8 = int8((num >> 16) & 0xff)
	amf0Encoder.writer.WriteByte(byte(tmp8))
	tmp8 = int8((num >> 8) & 0xff)
	amf0Encoder.writer.WriteByte(byte(tmp8))
	tmp8 = int8((num >> 0) & 0xff)
	amf0Encoder.writer.WriteByte(byte(tmp8))

	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeInt32(num int32) (err error) {
	err = binary.Write(amf0Encoder.writer, binary.BigEndian, &num)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeUint32(num uint32) (err error) {
	err = binary.Write(amf0Encoder.writer, binary.BigEndian, &num)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeInt32LittleEndian(num int32) (err error) {
	err = binary.Write(amf0Encoder.writer, binary.LittleEndian, &num)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeNamedString(name string, str string) (err error) {
	amf0Encoder.EncodeInt16(int16(len(name)))
	amf0Encoder.writer.Write([]byte(name))

	err = amf0Encoder.EncodeString(str)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeNamedBool(name string, boolean bool) (err error) {
	amf0Encoder.EncodeInt16(int16(len(name)))
	amf0Encoder.writer.Write([]byte(name))

	err = amf0Encoder.EncodeBool(boolean)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) EncodeNamedNumber(name string, num float64) (err error) {
	amf0Encoder.EncodeInt16(int16(len(name)))
	amf0Encoder.writer.Write([]byte(name))

	err = amf0Encoder.EncodeNumber(num)
	if err != nil {
		return err
	}
	return err
}

func (amf0Encoder *AMF0Encoder) AppendByteArray(data []byte) (err error) {
	_, err = amf0Encoder.writer.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (amf0Encoder *AMF0Encoder) AppendByte(data byte) (err error) {
	err = amf0Encoder.writer.WriteByte(data)
	return err
}

func (amf0Encoder *AMF0Encoder) GetData() (data []byte, err error) {
	if amf0Encoder.writer == nil {
		return nil, errors.New("not inited amf0 encoder")
	}
	return amf0Encoder.writer.Bytes(), nil
}

func (amf0Encoder *AMF0Encoder) GetDataSize() int {
	return len(amf0Encoder.writer.Bytes())
}

func (amf0Encoder *AMF0Encoder) EncodeAMFObj(obj *AMF0Object) {
	for v := obj.Props.Front(); v != nil; v = v.Next() {
		ret := amf0Encoder.encodeProp(v.Value.(*AMF0Property))
		if len(ret) > 0 {
			amf0Encoder.writer.Write(ret)
		}
	}
	return
}

func (amf0Encoder *AMF0Encoder) encodeObj(obj *AMF0Object) (data []byte) {
	//each prop +obj_end
	enc := &AMF0Encoder{}
	enc.Init()
	for v := obj.Props.Front(); v != nil; v = v.Next() {
		enc.AppendByteArray(amf0Encoder.encodeProp(v.Value.(*AMF0Property)))
	}
	enc.EncodeInt24(TAMF0EndOfObject)
	data, _ = enc.GetData()
	return data
}

func (amf0Encoder *AMF0Encoder) encodeProp(prop *AMF0Property) (data []byte) {
	enc := &AMF0Encoder{}
	enc.Init()

	//has name
	if len(prop.Name) > 0 {
		enc.EncodeString(prop.Name)
	}
	//encode type
	switch prop.PropType {
	case TAMF0Number:
		enc.EncodeNumber(prop.Value.NumValue)
	case TAMF0Boolean:
		enc.EncodeBool(prop.Value.BoolValue)
	case TAMF0String:
		enc.EncodeString(prop.Value.StrValue)
	case TAMF0Object:
		enc.AppendByte(TAMF0Object)
		enc.AppendByteArray(amf0Encoder.encodeObj(&prop.Value.ObjValue))
	case TAMF0Null:
		enc.AppendByte(TAMF0Null)
	case TAMF0EcmaArray:
		enc.AppendByte(TAMF0EcmaArray)
		//size
		//object
		tmp := enc.encodeObj(&prop.Value.ObjValue)
		tmpSize := int32(len(tmp))
		enc.EncodeInt32(tmpSize)
		enc.AppendByteArray(tmp)
	case TAMF0StrictArray:
		enc.AppendByte(TAMF0StrictArray)
		enc.EncodeInt32(int32(prop.Value.ObjValue.Props.Len()))
		for v := prop.Value.ObjValue.Props.Front(); v != nil; v = v.Next() {
			enc.AppendByteArray(amf0Encoder.encodeProp(v.Value.(*AMF0Property)))
		}

	case TAMF0Date:
		enc.AppendByte(TAMF0Date)
		binary.Write(enc.writer, binary.BigEndian, &prop.Value.NumValue)
		enc.EncodeInt16(prop.Value.S16Value)
	case TAMF0LongString:
		enc.EncodeString(prop.Value.StrValue)
	default:
		logger.LOGW(fmt.Sprintf("not support amf type:%d", prop.PropType))
	}
	data, _ = enc.GetData()
	return
}

//解码
func AMF0DecodeInt16(data []byte) (ret uint16, err error) {
	err = binary.Read(bytes.NewReader(data), binary.BigEndian, &ret)
	return ret, err
}

func AMF0DecodeInt24(data []byte) (ret uint32, err error) {
	ret = ((uint32(data[0])) << 16) | ((uint32(data[1])) << 8) | ((uint32(data[2])) << 0)
	return ret, nil
}

func AMF0DecodeInt32(data []byte) (ret uint32, err error) {
	err = binary.Read(bytes.NewReader(data), binary.BigEndian, &ret)
	return ret, err
}

func AMF0DecodeInt32LE(data []byte) (ret uint32, err error) {
	err = binary.Read(bytes.NewReader(data), binary.LittleEndian, &err)
	return ret, err
}

func AMF0DecodeNumber(data []byte) (ret float64, err error) {
	err = binary.Read(bytes.NewReader(data), binary.BigEndian, &ret)
	return ret, err
}

func AMF0DecodeBoolean(data []byte) (ret bool, err error) {
	if data[0] == 0 {
		ret = false
	} else {
		ret = true
	}
	return ret, nil
}

func AMF0DecodeString(data []byte) (ret string, err error) {
	strLength, err := AMF0DecodeInt16(data)
	if err != nil {
		return ret, err
	}
	tmpArray := make([]byte, strLength)
	_, err = bytes.NewReader(data[2:]).Read(tmpArray)
	ret = string(tmpArray)
	return ret, err
}

func AMFDecodeLongString(data []byte) (ret string, err error) {
	strLength, err := AMF0DecodeInt32(data)
	if err != nil {
		return ret, err
	}
	tmpArray := make([]byte, strLength)
	_, err = bytes.NewReader(data[4:]).Read(tmpArray)
	ret = string(tmpArray)
	return ret, err
}

func AMF0DecodeObj(data []byte) (ret *AMF0Object, err error) {
	ret, _, err = amf0DecodeObj(data, false)
	return ret, err
}

func amf0DecodeObj(data []byte, decodeName bool) (ret *AMF0Object, sizeUsed int32, err error) {
	ret = &AMF0Object{}
	var start, end int32
	start = 0
	end = int32(len(data))
	for start < end {
		if end-start >= 3 {
			endType, _ := AMF0DecodeInt24(data[start:])
			if endType == TAMF0EndOfObject {
				start += 3
				err = nil
				break
			}
		}
		if err != nil {
			break
		}
		propGeted, bytesUsed, err := amf0DecodeProp(data[start:], decodeName)
		if err != nil {
			break
		}
		if propGeted == nil || bytesUsed < 1 {
			break
		} else {
			start += bytesUsed
			ret.Props.PushBack(propGeted)
		}

	}
	if err != nil {
		return nil, -1, err
	}

	sizeUsed = start

	return ret, sizeUsed, err
}

func amf0DecodeProp(data []byte, decodeName bool) (ret *AMF0Property, sizeUsed int32, err error) {
	ret = &AMF0Property{}
	sizeUsed = 0
	err = nil

	if decodeName {
		if len(data) < 4 {
			return ret, sizeUsed, errors.New("no enough data for decode name")
		}
		nameSize, err := AMF0DecodeInt16(data[sizeUsed:])
		if err != nil {
			return ret, sizeUsed, nil
		}
		name, err := AMF0DecodeString(data[sizeUsed:])
		if err != nil {
			return ret, sizeUsed, nil
		}
		ret.Name = name
		sizeUsed += int32(2 + nameSize)
	}

	ret.PropType = int32(data[sizeUsed])
	sizeUsed += 1

	switch ret.PropType {
	case TAMF0Number:
		err = binary.Read(bytes.NewReader(data[sizeUsed:]), binary.BigEndian, &ret.Value.NumValue)
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += 8
	case TAMF0Boolean:
		if data[sizeUsed] == 0 {
			ret.Value.BoolValue = false
		} else {
			ret.Value.BoolValue = true
		}
		sizeUsed += 1
	case TAMF0String:
		stringLength, err := AMF0DecodeInt16(data[sizeUsed:])
		if err != nil {
			return ret, sizeUsed, err
		}
		if stringLength == 0 {
			ret.Value.StrValue = ""
		} else {
			ret.Value.StrValue, err = AMF0DecodeString(data[sizeUsed:])
			if err != nil {
				return ret, sizeUsed, err
			}
		}
		sizeUsed += int32(2 + stringLength)
	case TAMF0Object:
		tmpObj, size, err := amf0DecodeObj(data[sizeUsed:], true)
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += size
		ret.Value.ObjValue = *tmpObj
	case TAMF0Null:
	case TAMF0EcmaArray:
		sizeUsed += 4
		tmpObj, size, err := amf0DecodeObj(data[sizeUsed:], true)
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += size
		ret.Value.ObjValue = *tmpObj
	case TAMF0StrictArray:
		size, err := amf0ReadStrictArray(data[sizeUsed:], ret)
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += size
	case TAMF0Date:
		ret.Value.NumValue, err = AMF0DecodeNumber(data[sizeUsed:])
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += 8
		tmpu16, err := (AMF0DecodeInt16(data[sizeUsed:]))
		ret.Value.S16Value = int16(tmpu16)
		if err != nil {
			return ret, sizeUsed, err
		}
		sizeUsed += 2
	case TAMF0LongString:
		stringLength, err := AMF0DecodeInt32(data[sizeUsed:])
		if err != nil {
			return ret, sizeUsed, err
		}
		if stringLength == 0 {
			ret.Value.StrValue = ""
		} else {
			ret.Value.StrValue, err = AMFDecodeLongString(data[sizeUsed:])
			if err != nil {
				return ret, sizeUsed, err
			}
		}
		sizeUsed += int32(4 + stringLength)
	default:
		err = errors.New(fmt.Sprintf("not support amf type:%d", ret.PropType))
	}

	return ret, sizeUsed, err
}

func amf0ReadStrictArray(data []byte, prop *AMF0Property) (sizeUsed int32, err error) {
	if prop == nil {
		return 0, errors.New("invalid prop in amfReadStrictArray")
	}
	var arrayCount uint32
	sizeUsed = 0
	arrayCount, err = AMF0DecodeInt32(data)
	if err != nil {
		return 0, err
	}
	sizeUsed += 4
	if arrayCount == 0 {
		return sizeUsed, err
	}
	for arrayCount > 0 {
		arrayCount--
		tmpProp, size, err := amf0DecodeProp(data[sizeUsed:], false)
		if err != nil {
			return sizeUsed, err
		}
		prop.Value.ObjValue.Props.PushBack(tmpProp)
		sizeUsed += size
	}

	return sizeUsed, err
}

func (amf0Encoder *AMF0Object) AMF0GetPropByIndex(index int) (prop *AMF0Property) {
	if index >= amf0Encoder.Props.Len() {
		return nil
	}
	i := 0
	for e := amf0Encoder.Props.Front(); e != nil; e = e.Next() {
		if i == index {
			prop = e.Value.(*AMF0Property)
			return prop
		}
		i++
	}
	return
}

func (amf0Encoder *AMF0Object) AMF0GetPropByName(name string) (prop *AMF0Property) {
	for e := amf0Encoder.Props.Front(); e != nil; e = e.Next() {
		v := e.Value.(*AMF0Property)
		if v.Name == name {
			prop = v
			return
		}
	}
	return nil
}

func (amf0Encoder *AMF0Object) Dump() {
	for e := amf0Encoder.Props.Front(); e != nil; e = e.Next() {
		prop := e.Value.(*AMF0Property)
		amf0Encoder.dumpProp(prop)
	}
}

func (amf0Encoder *AMF0Object) dumpProp(prop *AMF0Property) {
	if len(prop.Name) > 0 {
		logger.LOGT(prop.Name)
	}
	switch prop.PropType {
	case TAMF0EcmaArray:
		logger.LOGT("ecma array")
		amf0Encoder.dumpProp(prop)
	case TAMF0Object:
		logger.LOGT("object")

		prop.Value.ObjValue.Dump()
	case TAMF0StrictArray:
		logger.LOGT("static array")
		amf0Encoder.dumpProp(prop)
	case TAMF0String:
		logger.LOGT("string:" + prop.Value.StrValue)
	case TAMF0Number:
		logger.LOGT(prop.Value.NumValue)
	case TAMF0Boolean:
		logger.LOGT(prop.Value.BoolValue)
	}

}
