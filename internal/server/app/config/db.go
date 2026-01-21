package config

type DBConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxConns         int32  `mapstructure:"max_conns"`
	MinConns         int32  `mapstructure:"min_conns"`
}
