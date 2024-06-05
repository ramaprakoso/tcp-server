package teltonika_decoder

import (
	"fmt"

	"github.com/filipkroca/b2n"
)

type TCP_Packet struct {
	Preamble       uint32     // 0x00000000 preamble for TCP packet
	Avl_data_len   uint32     // size is calculated starting from Codec ID to Number of Data 2
	Codec_id       uint8      // 0x08 (codec 8)
	Avl_data_count uint8      // number of records in the packet
	Avl_data       []AVL_Data // slice with avl data
	CRC_16         int32
	Response       []byte
}

type AVL_Data struct {
	Timestamp     uint64 // a difference, in milliseconds, between the current time and midnight, January, 1970 UTC (UNIX time)
	Priority      uint8  // priority, [0 Low, 1 High, 2 Panic]
	Longitude     int32  //
	Latitude      int32
	Altitude      int32
	Angle         int32
	Satellite     int8
	Speed         uint16
	Event_id      uint8
	Element_count uint8
	IO_elements   []IO_Element
}

type IO_Element struct {
	Element_len uint8
	Element_id  uint8
	Element_val []byte
}

func Decode(bs *[]byte) (TCP_Packet, error) {
	decoded := TCP_Packet{}
	var err error
	var next_byte int

	// check for minimum packet size
	if len(*bs) < 45 {
		return TCP_Packet{}, fmt.Errorf("minimum packet size is 45 bytes, got %v", len(*bs))
	}

	// parse preamble
	decoded.Preamble, err = b2n.ParseBs2Uint32(bs, 0)
	if err != nil {
		return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
	}
	// check for preamble - the packet starts with four zero bytes
	if decoded.Preamble != 0 {
		return TCP_Packet{}, fmt.Errorf("the packet doesn't starts with four zero bytes, probably not teltonika packet, trashed")
	}

	// parse avl data length
	decoded.Avl_data_len, err = b2n.ParseBs2Uint32(bs, 4)
	if err != nil {
		return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
	}

	// parse codec id
	decoded.Codec_id = (*bs)[8]
	if decoded.Codec_id != 0x08 {
		return TCP_Packet{}, fmt.Errorf("invalid codec id, want 0x08 or 0x8e, get %v", decoded.Codec_id)
	}

	// parse number of data
	decoded.Avl_data_count, err = b2n.ParseBs2Uint8(bs, 9)
	if err != nil {
		return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
	}

	// make slice for avl data
	decoded.Avl_data = make([]AVL_Data, 0, decoded.Avl_data_count)

	// initialize next_byte counter
	next_byte = 10

	// go trough avl data
	for i := 0; i < int(decoded.Avl_data_count); i++ {
		decoded_avl_data := AVL_Data{}

		// parse timestamp
		decoded_avl_data.Timestamp, err = b2n.ParseBs2Uint64(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 8

		// parse priority
		decoded_avl_data.Priority, err = b2n.ParseBs2Uint8(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}
		// check priority value
		if decoded_avl_data.Priority > 2 {
			return TCP_Packet{}, fmt.Errorf("invalid Priority value, want Priority <= 2, got %v", decoded_avl_data.Priority)
		}

		next_byte++

		// parse longitude
		decoded_avl_data.Longitude, err = b2n.ParseBs2Int32TwoComplement(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 4

		// parse latitude
		decoded_avl_data.Latitude, err = b2n.ParseBs2Int32TwoComplement(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 4

		// parse altitude
		decoded_avl_data.Altitude, err = b2n.ParseBs2Int32TwoComplement(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 2

		// parse angle
		decoded_avl_data.Angle, err = b2n.ParseBs2Int32TwoComplement(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 2

		// parse satellites
		decoded_avl_data.Satellite, err = b2n.ParseBs2Int8TwoComplement(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte++

		// parse speed
		decoded_avl_data.Speed, err = b2n.ParseBs2Uint16(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte += 2

		// parse event io id
		decoded_avl_data.Event_id, err = b2n.ParseBs2Uint8(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte++

		// parse element count - total number of properties coming with record (N = N1 + N2 + N4 + N8)
		decoded_avl_data.Element_count, err = b2n.ParseBs2Uint8(bs, next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode error, %v", err)
		}

		next_byte++

		// parse element ios
		decoded_io, end_byte, err := DecodeElements(bs, int(decoded_avl_data.Element_count), next_byte)
		if err != nil {
			return TCP_Packet{}, fmt.Errorf("decode io element error, %v", err)
		}

		next_byte = end_byte

		decoded_avl_data.IO_elements = decoded_io

		// append avl data slice
		decoded.Avl_data = append(decoded.Avl_data, decoded_avl_data)
	}

	// check number of parsed data
	if int(decoded.Avl_data_count) != len(decoded.Avl_data) {
		return TCP_Packet{}, fmt.Errorf("error when counting number of parsed data, want %v, got %v", int(decoded.Avl_data_count), len(decoded.Avl_data))
	}

	// check if packet was corretly parsed
	Avl_data_count_2 := (*bs)[next_byte]
	if decoded.Avl_data_count != Avl_data_count_2 {
		return TCP_Packet{}, fmt.Errorf("unexpected error: number of data 2 different with number of data 1, want %#x, got %#x", decoded.Avl_data_count, Avl_data_count_2)
	}

	next_byte++

	// parse crc-16 - calculated from Codec ID to the Second Number of Data
	decoded.CRC_16, err = b2n.ParseBs2Int32TwoComplement(bs, next_byte)
	if err != nil {
		return TCP_Packet{}, fmt.Errorf("decode crc-16 error, %v", err)
	}

	// parse response
	decoded.Response = []byte{0x00, 0x00, 0x00, (*bs)[9]}

	return decoded, nil
}
