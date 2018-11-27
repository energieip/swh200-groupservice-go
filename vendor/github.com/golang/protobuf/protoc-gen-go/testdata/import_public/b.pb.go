// Code generated by protoc-gen-go. DO NOT EDIT.
// source: import_public/b.proto

package import_public

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	sub "github.com/golang/protobuf/protoc-gen-go/testdata/import_public/sub"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Local struct {
	M                    *sub.M   `protobuf:"bytes,1,opt,name=m" json:"m,omitempty"`
	E                    *sub.E   `protobuf:"varint,2,opt,name=e,enum=goproto.test.import_public.sub.E" json:"e,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Local) Reset()         { *m = Local{} }
func (m *Local) String() string { return proto.CompactTextString(m) }
func (*Local) ProtoMessage()    {}
func (*Local) Descriptor() ([]byte, []int) {
	return fileDescriptor_84995586b3d09710, []int{0}
}

func (m *Local) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Local.Unmarshal(m, b)
}
func (m *Local) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Local.Marshal(b, m, deterministic)
}
func (m *Local) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Local.Merge(m, src)
}
func (m *Local) XXX_Size() int {
	return xxx_messageInfo_Local.Size(m)
}
func (m *Local) XXX_DiscardUnknown() {
	xxx_messageInfo_Local.DiscardUnknown(m)
}

var xxx_messageInfo_Local proto.InternalMessageInfo

func (m *Local) GetM() *sub.M {
	if m != nil {
		return m.M
	}
	return nil
}

func (m *Local) GetE() sub.E {
	if m != nil && m.E != nil {
		return *m.E
	}
	return sub.E_ZERO
}

func init() {
	proto.RegisterType((*Local)(nil), "goproto.test.import_public.Local")
}

func init() { proto.RegisterFile("import_public/b.proto", fileDescriptor_84995586b3d09710) }

var fileDescriptor_84995586b3d09710 = []byte{
	// 169 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xcd, 0xcc, 0x2d, 0xc8,
	0x2f, 0x2a, 0x89, 0x2f, 0x28, 0x4d, 0xca, 0xc9, 0x4c, 0xd6, 0x4f, 0xd2, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x17, 0x92, 0x4a, 0xcf, 0x07, 0x33, 0xf4, 0x4a, 0x52, 0x8b, 0x4b, 0xf4, 0x50, 0xd4, 0x48,
	0x49, 0xa2, 0x6a, 0x29, 0x2e, 0x4d, 0xd2, 0x4f, 0x84, 0x68, 0x53, 0xca, 0xe4, 0x62, 0xf5, 0xc9,
	0x4f, 0x4e, 0xcc, 0x11, 0xd2, 0xe7, 0x62, 0xcc, 0x95, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x36, 0x52,
	0xd4, 0xc3, 0x6d, 0x96, 0x5e, 0x71, 0x69, 0x92, 0x9e, 0x6f, 0x10, 0x63, 0x2e, 0x48, 0x43, 0xaa,
	0x04, 0x93, 0x02, 0xa3, 0x06, 0x1f, 0x61, 0x0d, 0xae, 0x41, 0x8c, 0xa9, 0x4e, 0x8e, 0x51, 0xf6,
	0xe9, 0x99, 0x25, 0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0xe9, 0xf9, 0x39, 0x89, 0x79,
	0xe9, 0xfa, 0x60, 0x6d, 0x49, 0xa5, 0x69, 0x10, 0x46, 0xb2, 0x6e, 0x7a, 0x6a, 0x9e, 0x6e, 0x7a,
	0xbe, 0x3e, 0xc8, 0x9c, 0x94, 0xc4, 0x92, 0x44, 0x7d, 0x14, 0xb3, 0x00, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x35, 0x0e, 0x6a, 0x82, 0xfc, 0x00, 0x00, 0x00,
}
