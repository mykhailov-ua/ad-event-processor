package telegram

import (
	"net/url"
)

const initDataMaxFields = 16

type initDataField struct {
	key string
	val string
}

func parseInitDataFields(raw string) (fields []initDataField, hash string, err error) {
	fields = make([]initDataField, 0, 4)
	for start := 0; start < len(raw); {
		end := start
		for end < len(raw) && raw[end] != '&' {
			end++
		}
		seg := raw[start:end]
		start = end
		if start < len(raw) {
			start++
		}
		if len(seg) == 0 {
			continue
		}
		eq := -1
		for i := range len(seg) {
			if seg[i] == '=' {
				eq = i
				break
			}
		}
		if eq < 0 {
			continue
		}
		key := seg[:eq]
		valEnc := seg[eq+1:]
		val, decErr := url.QueryUnescape(valEnc)
		if decErr != nil {
			return nil, "", decErr
		}
		if key == "hash" {
			hash = val
			continue
		}
		if key == "signature" {
			continue
		}
		if len(fields) >= initDataMaxFields {
			return nil, "", errInitDataTooManyFields
		}
		fields = append(fields, initDataField{key: key, val: val})
	}
	sortInitDataFields(fields)
	return fields, hash, nil
}

func sortInitDataFields(fields []initDataField) {
	for i := 1; i < len(fields); i++ {
		j := i
		for j > 0 && fields[j-1].key > fields[j].key {
			fields[j], fields[j-1] = fields[j-1], fields[j]
			j--
		}
	}
}

func appendInitDataCheckString(dst []byte, fields []initDataField) []byte {
	for i, f := range fields {
		if i > 0 {
			dst = append(dst, '\n')
		}
		dst = append(dst, f.key...)
		dst = append(dst, '=')
		dst = append(dst, f.val...)
	}
	return dst
}
