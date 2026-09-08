package wasmattest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var (
	ErrModuleTooLarge    = errors.New("wasmattest: module exceeds size cap")
	ErrImportForbidden   = errors.New("wasmattest: module imports forbidden")
	ErrExportMissing     = errors.New("wasmattest: required export missing")
	ErrMemoryTooLarge    = errors.New("wasmattest: memory exceeds page cap")
	ErrPoWNotFound       = errors.New("wasmattest: pow not found")
	ErrInvalidDifficulty = errors.New("wasmattest: invalid pow difficulty")
	ErrInvalidOffset     = errors.New("wasmattest: memory offset out of range")
)

type Limits struct {
	MaxModuleBytes uint32
	MaxMemoryPages uint32
	CallTimeout    time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxModuleBytes: MaxModuleBytes,
		MaxMemoryPages: MaxMemoryPages,
		CallTimeout:    250 * time.Millisecond,
	}
}

type Sandbox struct {
	rt      wazero.Runtime
	mod     api.Module
	mem     api.Memory
	closed  bool
	limits  Limits
	version uint32
	dataOff uint32
}

func VerifyModule(wasm []byte, lim Limits) error {
	if lim.MaxModuleBytes == 0 {
		lim = DefaultLimits()
	}
	if uint32(len(wasm)) > lim.MaxModuleBytes {
		return ErrModuleTooLarge
	}
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	mod, err := rt.Instantiate(ctx, wasm)
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	if err := checkExports(mod); err != nil {
		return err
	}
	mem := mod.Memory()
	if mem == nil {
		return ErrExportMissing
	}
	if mem.Size() > uint32(lim.MaxMemoryPages)*65536 {
		return ErrMemoryTooLarge
	}
	return nil
}

func LoadSandbox(ctx context.Context, wasm []byte, lim Limits) (*Sandbox, error) {
	if lim.MaxModuleBytes == 0 {
		lim = DefaultLimits()
	}
	if err := VerifyModule(wasm, lim); err != nil {
		return nil, err
	}
	rt := wazero.NewRuntime(ctx)
	mod, err := rt.Instantiate(ctx, wasm)
	if err != nil {
		rt.Close(ctx)
		return nil, err
	}
	mem := mod.Memory()
	if mem == nil {
		mod.Close(ctx)
		rt.Close(ctx)
		return nil, ErrExportMissing
	}
	ver, err := callU32(ctx, lim, mod, "aad_abi_version")
	if err != nil {
		mod.Close(ctx)
		rt.Close(ctx)
		return nil, err
	}
	dataOff, err := callU32(ctx, lim, mod, "aad_data_off")
	if err != nil {
		mod.Close(ctx)
		rt.Close(ctx)
		return nil, err
	}
	return &Sandbox{
		rt:      rt,
		mod:     mod,
		mem:     mem,
		limits:  lim,
		version: ver,
		dataOff: dataOff,
	}, nil
}

func (s *Sandbox) Close(ctx context.Context) error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	_ = s.mod.Close(ctx)
	return s.rt.Close(ctx)
}

func (s *Sandbox) ABIVersion() uint32 {
	if s == nil {
		return 0
	}
	return s.version
}

func (s *Sandbox) PoWSolve(ctx context.Context, salt []byte, difficulty uint8, maxTries uint32) (uint32, error) {
	if s == nil || s.closed {
		return 0, errors.New("wasmattest: sandbox closed")
	}
	if len(salt) != SaltLen {
		return 0, errors.New("wasmattest: salt length must be 16")
	}
	if difficulty == 0 || difficulty > MaxPoWDifficulty {
		return 0, ErrInvalidDifficulty
	}
	if maxTries == 0 {
		maxTries = DefaultMaxTries
	}
	const saltRel = 0
	saltOff := s.dataOff + saltRel
	if saltOff+SaltLen > s.mem.Size() {
		return 0, ErrInvalidOffset
	}
	if !s.mem.Write(saltOff, salt) {
		return 0, ErrInvalidOffset
	}
	nonce, err := callU32(ctx, s.limits, s.mod, "aad_pow_solve", uint64(saltRel), uint64(difficulty), 0, uint64(maxTries))
	if err != nil {
		return 0, err
	}
	if nonce == PoWNotFound {
		return 0, ErrPoWNotFound
	}
	return nonce, nil
}

func (s *Sandbox) FloatNoiseIEEE(ctx context.Context) ([]byte, error) {
	if s == nil || s.closed {
		return nil, errors.New("wasmattest: sandbox closed")
	}
	const hashRel = 64
	outOff := s.dataOff + hashRel
	if outOff+HashLen > s.mem.Size() {
		return nil, ErrInvalidOffset
	}
	status, err := callU32(ctx, s.limits, s.mod, "aad_float_noise_ieee", uint64(hashRel))
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return nil, fmt.Errorf("wasmattest: float_noise status %d", status)
	}
	out, ok := s.mem.Read(outOff, HashLen)
	if !ok {
		return nil, ErrInvalidOffset
	}
	buf := make([]byte, HashLen)
	copy(buf, out)
	return buf, nil
}

func (s *Sandbox) SHA256OneShot(ctx context.Context, msg []byte) ([]byte, error) {
	if s == nil || s.closed {
		return nil, errors.New("wasmattest: sandbox closed")
	}
	if len(msg) > MemSize {
		return nil, ErrInvalidOffset
	}
	const msgRel = 512
	const outRel = 1024
	msgOff := s.dataOff + msgRel
	outOff := s.dataOff + outRel
	if msgOff+uint32(len(msg)) > s.mem.Size() || outOff+HashLen > s.mem.Size() {
		return nil, ErrInvalidOffset
	}
	if !s.mem.Write(msgOff, msg) {
		return nil, ErrInvalidOffset
	}
	status, err := callU32(ctx, s.limits, s.mod, "aad_sha256_one_shot", uint64(msgRel), uint64(len(msg)), uint64(outRel))
	if err != nil {
		return nil, err
	}
	if status != 0 {
		return nil, fmt.Errorf("wasmattest: sha256_one_shot status %d", status)
	}
	out, ok := s.mem.Read(outOff, HashLen)
	if !ok {
		return nil, ErrInvalidOffset
	}
	buf := make([]byte, HashLen)
	copy(buf, out)
	return buf, nil
}

func (s *Sandbox) BenchMul(ctx context.Context, rounds uint32) (uint32, error) {
	if s == nil || s.closed {
		return 0, errors.New("wasmattest: sandbox closed")
	}
	return callU32(ctx, s.limits, s.mod, "aad_bench_mul", uint64(rounds))
}

func checkExports(mod api.Module) error {
	for _, name := range requiredExports {
		if mod.ExportedFunction(name) == nil && name != "memory" {
			return fmt.Errorf("%w: %s", ErrExportMissing, name)
		}
	}
	if mod.Memory() == nil {
		return fmt.Errorf("%w: memory", ErrExportMissing)
	}
	return nil
}

func callU32(parentCtx context.Context, lim Limits, mod api.Module, name string, args ...uint64) (uint32, error) {
	fn := mod.ExportedFunction(name)
	if fn == nil {
		return 0, fmt.Errorf("%w: %s", ErrExportMissing, name)
	}
	ctx := parentCtx
	if lim.CallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(parentCtx, lim.CallTimeout)
		defer cancel()
	}
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return uint32(results[0]), nil
}
