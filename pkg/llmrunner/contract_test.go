package llmrunner

import (
	"testing"

	"github.com/magomedcoder/gen-runner/pb/llmrunnerpb"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestContract_ChatMessage_hasNoToolProtoFields(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.ChatMessage{}).ProtoReflect().Descriptor()
	for _, name := range []string{"tool_call_id", "tool_name", "tool_calls_json"} {
		if d.Fields().ByName(protoreflect.Name(name)) != nil {
			t.Fatalf("у ChatMessage не должно быть поля %q", name)
		}
	}
}

func TestContract_SendMessageRequest_renderedPromptField(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.SendMessageRequest{}).ProtoReflect().Descriptor()
	f := d.Fields().ByName("rendered_prompt")
	if f == nil {
		t.Fatal("отсутствует SendMessageRequest.rendered_prompt")
	}

	if f.Kind() != protoreflect.StringKind {
		t.Fatalf("тип rendered_prompt=%v", f.Kind())
	}
}

func TestContract_SendMessageRequest_hasNoModelField(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.SendMessageRequest{}).ProtoReflect().Descriptor()
	if d.Fields().ByName("model") != nil {
		t.Fatal("поля SendMessageRequest.model быть не должно - сначала LoadModel")
	}
}

func TestContract_EmbedRequest_hasNoModelField(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.EmbedRequest{}).ProtoReflect().Descriptor()
	if d.Fields().ByName("model") != nil {
		t.Fatal("поля EmbedRequest.model быть не должно - сначала LoadModel")
	}
}

func TestContract_EmbedBatchRequest_hasNoModelField(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.EmbedBatchRequest{}).ProtoReflect().Descriptor()
	if d.Fields().ByName("model") != nil {
		t.Fatal("поля EmbedBatchRequest.model быть не должно - сначала LoadModel")
	}
}

func TestContract_GenerationParams_hasNoToolsField(t *testing.T) {
	t.Parallel()
	d := (&llmrunnerpb.GenerationParams{}).ProtoReflect().Descriptor()
	if d.Fields().ByName("tools") != nil {
		t.Fatal("GenerationParams.tools не должно быть в proto раннера")
	}
}
