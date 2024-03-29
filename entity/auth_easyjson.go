// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package entity

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson4a0f95aaDecodeGithubComJasonKhew96BiliroamingGoServerEntity(in *jlexer.Lexer, out *AccInfo) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "code":
			out.Code = int(in.Int())
		case "message":
			out.Message = string(in.String())
		case "data":
			easyjson4a0f95aaDecode(in, &out.Data)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson4a0f95aaEncodeGithubComJasonKhew96BiliroamingGoServerEntity(out *jwriter.Writer, in AccInfo) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"code\":"
		out.RawString(prefix[1:])
		out.Int(int(in.Code))
	}
	{
		const prefix string = ",\"message\":"
		out.RawString(prefix)
		out.String(string(in.Message))
	}
	{
		const prefix string = ",\"data\":"
		out.RawString(prefix)
		easyjson4a0f95aaEncode(out, in.Data)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v AccInfo) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson4a0f95aaEncodeGithubComJasonKhew96BiliroamingGoServerEntity(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v AccInfo) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson4a0f95aaEncodeGithubComJasonKhew96BiliroamingGoServerEntity(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *AccInfo) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson4a0f95aaDecodeGithubComJasonKhew96BiliroamingGoServerEntity(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *AccInfo) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson4a0f95aaDecodeGithubComJasonKhew96BiliroamingGoServerEntity(l, v)
}
func easyjson4a0f95aaDecode(in *jlexer.Lexer, out *struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
	VIP  struct {
		DueDate int64 `json:"due_date"`
	} `json:"vip"`
}) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "mid":
			out.Mid = int64(in.Int64())
		case "name":
			out.Name = string(in.String())
		case "vip":
			easyjson4a0f95aaDecode1(in, &out.VIP)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson4a0f95aaEncode(out *jwriter.Writer, in struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
	VIP  struct {
		DueDate int64 `json:"due_date"`
	} `json:"vip"`
}) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"mid\":"
		out.RawString(prefix[1:])
		out.Int64(int64(in.Mid))
	}
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix)
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"vip\":"
		out.RawString(prefix)
		easyjson4a0f95aaEncode1(out, in.VIP)
	}
	out.RawByte('}')
}
func easyjson4a0f95aaDecode1(in *jlexer.Lexer, out *struct {
	DueDate int64 `json:"due_date"`
}) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "due_date":
			out.DueDate = int64(in.Int64())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson4a0f95aaEncode1(out *jwriter.Writer, in struct {
	DueDate int64 `json:"due_date"`
}) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"due_date\":"
		out.RawString(prefix[1:])
		out.Int64(int64(in.DueDate))
	}
	out.RawByte('}')
}
