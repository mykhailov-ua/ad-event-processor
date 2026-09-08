package wasmattest

const (
	ABIVersion       = 1
	MemSize          = 4096
	SaltLen          = 16
	HashLen          = 32
	PoWNotFound      = 0xffffffff
	MaxPoWDifficulty = 4

	MaxModuleBytes  = 64 * 1024
	MaxMemoryPages  = 2 // 128 KiB wasm linear memory (linker-enforced)
	DefaultMaxTries = 2_000_000

	FloatNoiseIEEEHex = "06bad31060c1212ae832de4c031f7b31e3b48aed57858294478cb19450cf34ca"
)

var requiredExports = []string{
	"memory",
	"aad_abi_version",
	"aad_data_off",
	"aad_pow_solve",
	"aad_float_noise_ieee",
	"aad_sha256_one_shot",
	"aad_bench_mul",
}
