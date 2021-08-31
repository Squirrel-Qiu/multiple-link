package multiple_link

type mode int

const (
	ClientMode mode = iota
	ServerMode
	VERSION = 1
	defaultBufSize = 65536
)

type Config struct {
	Version int
	Mode mode
	BufSize int32
}

func DefaultConfig(m mode) *Config {
	return &Config{
		Version: 1,
		Mode: m,
		BufSize: defaultBufSize,
	}
}
