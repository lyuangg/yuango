package logging

// Config defines logging configuration.
type Config struct {
	Level    string `json:"level" yaml:"level"`
	Format   string `json:"format" yaml:"format"`
	Output   string `json:"output" yaml:"output"`
	Rotate   string `json:"rotate" yaml:"rotate"`
	MaxFiles int    `json:"max_files" yaml:"max_files"` // 最大保留的日志文件数量，0表示不限制
}