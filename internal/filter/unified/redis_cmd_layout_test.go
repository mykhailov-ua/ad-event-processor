package unified

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	redis "github.com/redis/go-redis/v9"
)

func TestRedisCmdHead_mirrorLayout_holdout(t *testing.T) {
	t.Helper()
	cmd := redis.NewCmd(context.Background(), "ping")
	cmdTyp := reflect.TypeOf(*cmd)
	if cmdTyp.NumField() != 2 {
		t.Fatalf("redis.Cmd field count = %d, want 2", cmdTyp.NumField())
	}
	baseField, ok := cmdTyp.FieldByName("baseCmd")
	if !ok {
		t.Fatal("redis.Cmd missing baseCmd field")
	}
	valField, ok := cmdTyp.FieldByName("val")
	if !ok {
		t.Fatal("redis.Cmd missing val field")
	}
	headSize := valField.Offset + valField.Type.Size()
	mirrorSize := unsafe.Sizeof(redisCmdHead{})
	if headSize != mirrorSize {
		t.Fatalf("redisCmdHead size %d != Cmd size %d (go-redis layout drift)", mirrorSize, headSize)
	}
	if baseField.Offset != 0 {
		t.Fatalf("baseCmd offset = %d, want 0", baseField.Offset)
	}
	if valField.Offset != unsafe.Offsetof(redisCmdHead{}.val) {
		t.Fatalf("redis.Cmd.val offset %d != redisCmdHead.val offset %d", valField.Offset, unsafe.Offsetof(redisCmdHead{}.val))
	}

	headTyp := reflect.TypeOf(redisCmdHead{})
	baseTyp := baseField.Type
	if headTyp.NumField() != baseTyp.NumField()+1 {
		t.Fatalf("redisCmdHead fields %d != baseCmd fields %d + val", headTyp.NumField(), baseTyp.NumField())
	}
	for i := range baseTyp.NumField() {
		hf := headTyp.Field(i)
		bf := baseTyp.Field(i)
		if hf.Name != bf.Name || hf.Offset != bf.Offset || hf.Type != bf.Type {
			t.Fatalf("field %q mismatch head off=%d type=%v base off=%d type=%v", bf.Name, hf.Offset, hf.Type, bf.Offset, bf.Type)
		}
	}
	if headTyp.Field(baseTyp.NumField()).Name != "val" {
		t.Fatalf("last redisCmdHead field want val, got %q", headTyp.Field(baseTyp.NumField()).Name)
	}
}

func TestResetPooledRedisCmd_holdoutClearsVal(t *testing.T) {
	cmd := redis.NewCmd(context.Background(), "evalsha", "sha", 1, "k")
	cmd.SetVal(int64(99))
	resetPooledRedisCmd(cmd, context.Background(), []any{"evalsha", "h", 1, "k2"}, 3)
	h := (*redisCmdHead)(unsafe.Pointer(cmd))
	if h.val != nil {
		t.Fatal("holdout: resetPooledRedisCmd must clear val to avoid stale pooled Cmd reads")
	}
}
