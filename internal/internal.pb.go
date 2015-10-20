// Code generated by protoc-gen-gogo.
// source: internal/internal.proto
// DO NOT EDIT!

/*
Package internal is a generated protocol buffer package.

It is generated from these files:
	internal/internal.proto

It has these top-level messages:
	Repository
	Message
*/
package internal

import proto "github.com/gogo/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Repository struct {
	ID               *string    `protobuf:"bytes,1,req" json:"ID,omitempty"`
	Description      *string    `protobuf:"bytes,2,req" json:"Description,omitempty"`
	Language         *string    `protobuf:"bytes,3,req" json:"Language,omitempty"`
	Notified         *bool      `protobuf:"varint,4,req" json:"Notified,omitempty"`
	Messages         []*Message `protobuf:"bytes,5,rep" json:"Messages,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *Repository) Reset()         { *m = Repository{} }
func (m *Repository) String() string { return proto.CompactTextString(m) }
func (*Repository) ProtoMessage()    {}

func (m *Repository) GetID() string {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return ""
}

func (m *Repository) GetDescription() string {
	if m != nil && m.Description != nil {
		return *m.Description
	}
	return ""
}

func (m *Repository) GetLanguage() string {
	if m != nil && m.Language != nil {
		return *m.Language
	}
	return ""
}

func (m *Repository) GetNotified() bool {
	if m != nil && m.Notified != nil {
		return *m.Notified
	}
	return false
}

func (m *Repository) GetMessages() []*Message {
	if m != nil {
		return m.Messages
	}
	return nil
}

type Message struct {
	ID               *uint64 `protobuf:"varint,1,req" json:"ID,omitempty"`
	Text             *string `protobuf:"bytes,2,req" json:"Text,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}

func (m *Message) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Message) GetText() string {
	if m != nil && m.Text != nil {
		return *m.Text
	}
	return ""
}

func init() {
}