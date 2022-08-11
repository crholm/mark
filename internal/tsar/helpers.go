package tsar

import "encoding/binary"

func bytesOfUint32(u uint32) []byte{
	d := []byte{0,0,0,0}
	binary.BigEndian.PutUint32(d, u)
	return d
}
func uint32OfBytes(b []byte) uint32{
	return binary.BigEndian.Uint32(b)
}
func bytesOfUint16(u uint16) []byte{
	d := []byte{0,0}
	binary.BigEndian.PutUint16(d, u)
	return d
}
func uint16OfBytes(b []byte) uint16{
	return binary.BigEndian.Uint16(b)
}