package teltonika_decoder

import (
	"fmt"

	"github.com/filipkroca/b2n"
)

func DecodeElements(bs *[]byte, element_count int, start_byte int) ([]IO_Element, int, error) {
	element_checksum := 0

	// make slice for io element
	element_slice := make([]IO_Element, 0, element_count)

	next_byte := start_byte

	// parse N1 - number of properties, which length is 1 byte
	x, err := b2n.ParseBs2Uint8(bs, next_byte)
	if err != nil {
		return []IO_Element{}, 0, fmt.Errorf("decode io elements 1 byte error %v", err)
	}
	n_of_elements := int(x)

	next_byte++

	// parse 1 byte ios
	for i := 0; i < n_of_elements; i++ {
		cutted, err := CutIO(bs, next_byte, 1)
		if err != nil {
			return []IO_Element{}, 0, fmt.Errorf("decode io elements 1 byte error %v", err)
		}

		//append element to the returned slice
		element_slice = append(element_slice, cutted)
		next_byte += 2
		element_checksum++
	}

	// parse N2 - number of properties, which length is 2 bytes
	x, err = b2n.ParseBs2Uint8(bs, next_byte)
	if err != nil {
		return []IO_Element{}, 0, fmt.Errorf("decode io elements 2 bytes error %v", err)
	}
	n_of_elements = int(x)

	next_byte++

	// parse 2 bytes ios
	for i := 0; i < n_of_elements; i++ {
		cutted, err := CutIO(bs, next_byte, 2)
		if err != nil {
			return []IO_Element{}, 0, fmt.Errorf("decode io elements 2 bytes error %v", err)
		}

		//append element to the returned slice
		element_slice = append(element_slice, cutted)
		next_byte += 3
		element_checksum++
	}

	// parse N4 - number of properties, which length is 4 bytes
	x, err = b2n.ParseBs2Uint8(bs, next_byte)
	if err != nil {
		return []IO_Element{}, 0, fmt.Errorf("decode io elements 4 bytes error %v", err)
	}
	n_of_elements = int(x)

	next_byte++

	// parse 4 bytes ios
	for i := 0; i < n_of_elements; i++ {
		cutted, err := CutIO(bs, next_byte, 4)
		if err != nil {
			return []IO_Element{}, 0, fmt.Errorf("decode io elements 4 bytes error %v", err)
		}

		//append element to the returned slice
		element_slice = append(element_slice, cutted)
		next_byte += 5
		element_checksum++
	}

	// parse N8 - number of properties, which length is 8 bytes
	x, err = b2n.ParseBs2Uint8(bs, next_byte)
	if err != nil {
		return []IO_Element{}, 0, fmt.Errorf("decode io elements 8 bytes error %v", err)
	}
	n_of_elements = int(x)

	next_byte++

	// parse 8 bytes ios
	for i := 0; i < n_of_elements; i++ {
		cutted, err := CutIO(bs, next_byte, 8)
		if err != nil {
			return []IO_Element{}, 0, fmt.Errorf("decode io elements 8 bytes error %v", err)
		}

		//append element to the returned slice
		element_slice = append(element_slice, cutted)
		next_byte += 7
		element_checksum++
	}

	// check element check sum condition
	if element_checksum != element_count {
		//log.Fatalf("Error when counting parsed IO Elements, want %v, got %v", totalElements, totalElementsChecksum)
		return []IO_Element{}, 0, fmt.Errorf("error when counting parsed io elements, want %v, got %v", element_count, element_checksum)
	}
	return element_slice, next_byte, nil
}

func CutIO(bs *[]byte, start_byte int, length int) (IO_Element, error) {
	var err error
	// make slice for current io element
	current_io := IO_Element{}

	// parse current io element length
	current_io.Element_len = uint8(length)

	// parse current io element id
	current_io.Element_id, err = b2n.ParseBs2Uint8(bs, start_byte)
	if err != nil {
		return IO_Element{}, fmt.Errorf("cut io error, %v", err)
	}
	next_byte := start_byte + 1
	// parse current io element value
	current_io.Element_val = (*bs)[next_byte : next_byte+length]

	return current_io, nil
}
