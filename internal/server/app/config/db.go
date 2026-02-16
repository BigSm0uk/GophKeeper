package config

type DBConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	MaxConns         int32  `mapstructure:"max_conns"`
	MinConns         int32  `mapstructure:"min_conns"`
}

// NewDefaultDBConfig returns default database configuration
func NewDefaultDBConfig() DBConfig {
	return DBConfig{
		ConnectionString: "postgresql://user:password@localhost:5432/gophkeeper?sslmode=disable",
		MaxConns:         25,
		MinConns:         5,
	}
}
