package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		expectedPort string
		expectedTimer int
	}{
		{
			name:         "default port when PORT not set",
			envVars:      map[string]string{},
			expectedPort: "8000",
			expectedTimer: 60,
		},
		{
			name:         "uses PORT env var when set",
			envVars:      map[string]string{"PORT": "3000"},
			expectedPort: "3000",
			expectedTimer: 60,
		},
		{
			name:         "default refresh timer when not set",
			envVars:      map[string]string{},
			expectedPort: "8000",
			expectedTimer: 60,
		},
		{
			name:         "uses REFRESH_TIMER env var when set",
			envVars:      map[string]string{"REFRESH_TIMER": "120"},
			expectedPort: "8000",
			expectedTimer: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()
			
			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := LoadFromEnv()
			if err != nil {
				t.Fatalf("LoadFromEnv() error = %v", err)
			}

			if cfg.Server.Port != tt.expectedPort {
				t.Errorf("Port = %v, want %v", cfg.Server.Port, tt.expectedPort)
			}

			if cfg.Server.RefreshTimer != tt.expectedTimer {
				t.Errorf("RefreshTimer = %v, want %v", cfg.Server.RefreshTimer, tt.expectedTimer)
			}
		})
	}
}

func TestLoadFromEnv_ParsesRefreshTimerAsInt(t *testing.T) {
	os.Clearenv()
	os.Setenv("REFRESH_TIMER", "300")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Server.RefreshTimer != 300 {
		t.Errorf("RefreshTimer = %v, want %v", cfg.Server.RefreshTimer, 300)
	}
}

func TestLoadFromEnv_InvalidRefreshTimer(t *testing.T) {
	os.Clearenv()
	os.Setenv("REFRESH_TIMER", "not-a-number")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	// Should use default value when parsing fails
	if cfg.Server.RefreshTimer != 60 {
		t.Errorf("RefreshTimer = %v, want %v (default)", cfg.Server.RefreshTimer, 60)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Port:         "8000",
					RefreshTimer: 60,
				},
				Cache: CacheConfig{
					Type: "memory",
				},
			},
			wantErr: false,
		},
		{
			name: "empty port",
			config: Config{
				Server: ServerConfig{
					Port:         "",
					RefreshTimer: 60,
				},
				Cache: CacheConfig{
					Type: "memory",
				},
			},
			wantErr: true,
			errMsg:  "port cannot be empty",
		},
		{
			name: "refresh timer less than 1",
			config: Config{
				Server: ServerConfig{
					Port:         "8000",
					RefreshTimer: 0,
				},
				Cache: CacheConfig{
					Type: "memory",
				},
			},
			wantErr: true,
			errMsg:  "refresh timer must be at least 1 second",
		},
		{
			name: "invalid cache type",
			config: Config{
				Server: ServerConfig{
					Port:         "8000",
					RefreshTimer: 60,
				},
				Cache: CacheConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "cache type must be 'redis' or 'memory'",
		},
		{
			name: "redis type with empty address",
			config: Config{
				Server: ServerConfig{
					Port:         "8000",
					RefreshTimer: 60,
				},
				Cache: CacheConfig{
					Type: "redis",
					Redis: RedisConfig{
						Address: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "redis address cannot be empty when using redis cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}