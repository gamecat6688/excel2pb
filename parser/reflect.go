package parser

import (
	"context"
	"fmt"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// 解析 proto 文件并获取 FileDescriptor
func parseProtoFile(files []string, includePaths []string) (linker.Files, error) {
	compiler := &protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: includePaths,
		},
	}
	return compiler.Compile(context.Background(), files...)
}

// 获取消息类型描述符
func getMessageType(files linker.Files, fullName string) (protoreflect.MessageType, error) {
	desc, err := files.AsResolver().FindDescriptorByName(protoreflect.FullName(fullName))
	if err != nil {
		return nil, fmt.Errorf("找不到消息类型: %v", err)
	}

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("描述符不是消息类型")
	}

	return dynamicpb.NewMessageType(msgDesc), nil
}

// 动态设置字段值
func setDynamicFields(msg protoreflect.Message, fields map[string]interface{}) {
	descriptor := msg.Descriptor()

	for name, value := range fields {
		field := descriptor.Fields().ByName(protoreflect.Name(name))
		if field == nil {
			continue
		}

		switch {
		case field.IsList():
			list := msg.Mutable(field).List()
			for _, item := range value.([]interface{}) {
				list.Append(protoreflect.ValueOf(item))
			}
		case field.Kind() == protoreflect.MessageKind:
			nestedMsg := dynamicpb.NewMessage(field.Message())
			setDynamicFields(nestedMsg, value.(map[string]interface{}))
			msg.Set(field, protoreflect.ValueOfMessage(nestedMsg))
		default:
			msg.Set(field, protoreflect.ValueOf(value))
		}
	}
}
