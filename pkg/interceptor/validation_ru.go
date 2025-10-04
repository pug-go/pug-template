package interceptor

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	"buf.build/go/protovalidate"
	"github.com/AlekSi/pointer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type ErrorInfo struct {
	FieldName  string
	RuleValue  any
	FieldValue any
}

func UnaryServerValidationsRu(validator protovalidate.Validator) func(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := validateMsg(req, validator); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func StreamServerValidationsRu(validator protovalidate.Validator) func(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &wrappedServerStream{
			ServerStream: stream,
			validator:    validator,
		})
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	validator protovalidate.Validator
}

func (w *wrappedServerStream) RecvMsg(m interface{}) error {
	if err := w.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	return validateMsg(m, w.validator)
}

func validateMsg(m interface{}, validator protovalidate.Validator) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return status.Errorf(codes.Internal, "unsupported message type: %T", m)
	}
	err := validator.Validate(msg)
	if err == nil {
		return nil
	}

	var valErr *protovalidate.ValidationError
	if errors.As(err, &valErr) {
		for _, violation := range valErr.Violations {
			ruleID := violation.Proto.GetRuleId()
			tmpl, ok := ruTemplates[ruleID]
			if !ok {
				continue
			}

			t, terr := template.New(ruleID).Parse(tmpl)
			if terr != nil {
				continue
			}

			var buf bytes.Buffer
			_ = t.Execute(&buf, ErrorInfo{
				FieldName:  violation.Proto.GetField().GetElements()[0].GetFieldName(),
				RuleValue:  violation.RuleValue.Interface(),
				FieldValue: violation.FieldValue.Interface(),
			})

			// overwrite the violation message with our localized/rendered text
			violation.Proto.Message = pointer.ToString(buf.String())
		}

		st := status.New(codes.InvalidArgument, err.Error())
		ds, detErr := st.WithDetails(valErr.ToProto())
		if detErr != nil {
			return st.Err()
		}
		return ds.Err()
	}

	// CEL expression doesn't compile or type-check.
	return status.Error(codes.Internal, err.Error())
}

var ruTemplates = map[string]string{
	"float.const":  "значение должно быть равно {{.RuleValue}}",
	"float.in":     "значение должно входить в список {{.RuleValue}}",
	"float.not_in": "значение не должно входить в список {{.RuleValue}}",
	"float.finite": "значение {{.FieldValue}} должно быть конечным числом",

	"double.const":  "значение должно быть равно {{.RuleValue}}",
	"double.in":     "значение должно входить в список {{.RuleValue}}",
	"double.not_in": "значение не должно входить в список {{.RuleValue}}",
	"double.finite": "значение {{.FieldValue}} должно быть конечным числом",

	"int32.const":  "значение должно быть равно {{.RuleValue}}",
	"int32.in":     "значение должно входить в список {{.RuleValue}}",
	"int32.not_in": "значение не должно входить в список {{.RuleValue}}",

	"int64.const":  "значение должно быть равно {{.RuleValue}}",
	"int64.in":     "значение должно входить в список {{.RuleValue}}",
	"int64.not_in": "значение не должно входить в список {{.RuleValue}}",

	"uint32.const":  "значение должно быть равно {{.RuleValue}}",
	"uint32.in":     "значение должно входить в список {{.RuleValue}}",
	"uint32.not_in": "значение не должно входить в список {{.RuleValue}}",

	"uint64.const":  "значение должно быть равно {{.RuleValue}}",
	"uint64.in":     "значение должно входить в список {{.RuleValue}}",
	"uint64.not_in": "значение не должно входить в список {{.RuleValue}}",

	"sint32.const":  "значение должно быть равно {{.RuleValue}}",
	"sint32.in":     "значение должно входить в список {{.RuleValue}}",
	"sint32.not_in": "значение не должно входить в список {{.RuleValue}}",

	"sint64.const":  "значение должно быть равно {{.RuleValue}}",
	"sint64.in":     "значение должно входить в список {{.RuleValue}}",
	"sint64.not_in": "значение не должно входить в список {{.RuleValue}}",

	"fixed32.const":  "значение должно быть равно {{.RuleValue}}",
	"fixed32.in":     "значение должно входить в список {{.RuleValue}}",
	"fixed32.not_in": "значение не должно входить в список {{.RuleValue}}",

	"fixed64.const":  "значение должно быть равно {{.RuleValue}}",
	"fixed64.in":     "значение должно входить в список {{.RuleValue}}",
	"fixed64.not_in": "значение не должно входить в список {{.RuleValue}}",

	"sfixed32.const":  "значение должно быть равно {{.RuleValue}}",
	"sfixed32.in":     "значение должно входить в список {{.RuleValue}}",
	"sfixed32.not_in": "значение не должно входить в список {{.RuleValue}}",

	"sfixed64.const":  "значение должно быть равно {{.RuleValue}}",
	"sfixed64.in":     "значение должно входить в список {{.RuleValue}}",
	"sfixed64.not_in": "значение не должно входить в список {{.RuleValue}}",

	"bool.const": "значение должно быть равно {{.RuleValue}}",

	"string.const":        "значение должно быть равно «{{.RuleValue}}»",
	"string.len":          "длина значения должна быть {{.RuleValue}} символов",
	"string.min_len":      "длина значения должна быть не меньше {{.RuleValue}} символов",
	"string.max_len":      "длина значения должна быть не больше {{.RuleValue}} символов",
	"string.len_bytes":    "длина значения должна быть {{.RuleValue}} байт",
	"string.min_bytes":    "длина значения должна быть не меньше {{.RuleValue}} байт",
	"string.max_bytes":    "длина значения должна быть не больше {{.RuleValue}} байт",
	"string.pattern":      "значение не соответствует шаблону регулярного выражения «{{.RuleValue}}»",
	"string.prefix":       "значение не имеет префикса «{{.RuleValue}}»",
	"string.suffix":       "значение не имеет суффикса «{{.RuleValue}}»",
	"string.contains":     "значение не содержит подстроку «{{.RuleValue}}»",
	"string.not_contains": "значение содержит запрещённую подстроку «{{.RuleValue}}»",
	"string.in":           "значение должно входить в список {{.RuleValue}}",
	"string.not_in":       "значение не должно входить в список {{.RuleValue}}",

	"string.email":                              "значение должно быть корректным адресом электронной почты",
	"string.email_empty":                        "пустое значение не является допустимым адресом электронной почты",
	"string.hostname":                           "значение должно быть корректным именем хоста",
	"string.hostname_empty":                     "пустое значение не является допустимым именем хоста",
	"string.ip":                                 "значение должно быть корректным IP-адресом",
	"string.ip_empty":                           "пустое значение не является допустимым IP-адресом",
	"string.ipv4":                               "значение должно быть корректным IPv4-адресом",
	"string.ipv4_empty":                         "пустое значение не является допустимым IPv4-адресом",
	"string.ipv6":                               "значение должно быть корректным IPv6-адресом",
	"string.ipv6_empty":                         "пустое значение не является допустимым IPv6-адресом",
	"string.uri":                                "значение должно быть корректным URI",
	"string.uri_empty":                          "пустое значение не является допустимым URI",
	"string.uri_ref":                            "значение должно быть корректной ссылкой URI",
	"string.address":                            "значение должно быть корректным именем хоста или IP-адресом",
	"string.address_empty":                      "пустое значение не является допустимым именем хоста или IP-адресом",
	"string.uuid":                               "значение должно быть корректным UUID",
	"string.uuid_empty":                         "пустое значение не является допустимым UUID",
	"string.tuuid":                              "значение должно быть корректным UUID без дефисов",
	"string.tuuid_empty":                        "пустое значение не является допустимым UUID без дефисов",
	"string.ip_with_prefixlen":                  "значение должно быть корректным IP с длиной префикса",
	"string.ip_with_prefixlen_empty":            "пустое значение не является допустимым IP с длиной префикса",
	"string.ipv4_with_prefixlen":                "значение должно быть корректным IPv4 с длиной префикса",
	"string.ipv4_with_prefixlen_empty":          "пустое значение не является допустимым IPv4 с длиной префикса",
	"string.ipv6_with_prefixlen":                "значение должно быть корректным IPv6 с длиной префикса",
	"string.ipv6_with_prefixlen_empty":          "пустое значение не является допустимым IPv6 с длиной префикса",
	"string.ip_prefix":                          "значение должно быть корректным IP-префиксом",
	"string.ip_prefix_empty":                    "пустое значение не является допустимым IP-префиксом",
	"string.ipv4_prefix":                        "значение должно быть корректным IPv4-префиксом",
	"string.ipv4_prefix_empty":                  "пустое значение не является допустимым IPv4-префиксом",
	"string.ipv6_prefix":                        "значение должно быть корректным IPv6-префиксом",
	"string.ipv6_prefix_empty":                  "пустое значение не является допустимым IPv6-префиксом",
	"string.host_and_port":                      "значение должно быть корректной парой «хост:порт»",
	"string.host_and_port_empty":                "пустое значение не является допустимой парой «хост:порт»",
	"string.well_known_regex.header_name":       "значение должно быть корректным именем HTTP-заголовка",
	"string.well_known_regex.header_name_empty": "пустое значение не является допустимым именем HTTP-заголовка",
	"string.well_known_regex.header_value":      "значение должно быть корректным значением HTTP-заголовка",

	"bytes.const":      "значение должно быть равно {{.RuleValue}}",
	"bytes.len":        "длина значения должна быть {{.RuleValue}} байт",
	"bytes.min_len":    "длина значения должна быть не меньше {{.RuleValue}} байт",
	"bytes.max_len":    "длина значения должна быть не больше {{.RuleValue}} байт",
	"bytes.pattern":    "значение должно соответствовать шаблону «{{.RuleValue}}»",
	"bytes.prefix":     "значение не имеет префикса {{.RuleValue}}",
	"bytes.suffix":     "значение не имеет суффикса {{.RuleValue}}",
	"bytes.contains":   "значение не содержит {{.RuleValue}}",
	"bytes.in":         "значение должно входить в список {{.RuleValue}}",
	"bytes.not_in":     "значение не должно входить в список {{.RuleValue}}",
	"bytes.ip":         "значение должно быть корректным IP-адресом",
	"bytes.ip_empty":   "пустое значение не является допустимым IP-адресом",
	"bytes.ipv4":       "значение должно быть корректным IPv4-адресом",
	"bytes.ipv4_empty": "пустое значение не является допустимым IPv4-адресом",
	"bytes.ipv6":       "значение должно быть корректным IPv6-адресом",
	"bytes.ipv6_empty": "пустое значение не является допустимым IPv6-адресом",

	"enum.const":  "значение должно быть равно {{.RuleValue}}",
	"enum.in":     "значение должно входить в список {{.RuleValue}}",
	"enum.not_in": "значение не должно входить в список {{.RuleValue}}",

	"repeated.min_items": "в списке должно быть не менее {{.RuleValue}} элементов",
	"repeated.max_items": "в списке должно быть не более {{.RuleValue}} элементов",
	"repeated.unique":    "значения в списке должны быть уникальными",

	"map.min_pairs": "в карте должно быть не менее {{.RuleValue}} пар ключ-значение",
	"map.max_pairs": "в карте должно быть не более {{.RuleValue}} пар ключ-значение",

	"duration.const":  "значение должно быть равно {{.RuleValue}}",
	"duration.in":     "значение должно входить в список {{.RuleValue}}",
	"duration.not_in": "значение не должно входить в список {{.RuleValue}}",

	"timestamp.const":  "значение должно быть равно {{.RuleValue}}",
	"timestamp.lt_now": "значение должно быть меньше текущего времени",
	"timestamp.gt_now": "значение должно быть больше текущего времени",
	"timestamp.within": "значение должно находиться в пределах {{.RuleValue}} от текущего времени",
}
