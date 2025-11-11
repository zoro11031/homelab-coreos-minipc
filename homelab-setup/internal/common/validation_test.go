package common

import "testing"

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv4 with zeros", "10.0.0.1", false},
		{"invalid - too high", "256.1.1.1", true},
		{"invalid - not numeric", "not-an-ip", true},
		{"invalid - empty", "", true},
		{"invalid - IPv6", "2001:0db8:85a3::8a2e:0370:7334", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"valid port", "8080", false},
		{"valid port - min", "1", false},
		{"valid port - max", "65535", false},
		{"invalid - zero", "0", true},
		{"invalid - too high", "65536", true},
		{"invalid - negative", "-1", true},
		{"invalid - not numeric", "abc", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid absolute path", "/home/user/config", false},
		{"valid root", "/", false},
		{"invalid - relative", "relative/path", true},
		{"invalid - relative dot", "./path", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "johndoe", false},
		{"valid with underscore", "_system", false},
		{"valid with hyphen", "john-doe", false},
		{"valid with numbers", "user123", false},
		{"invalid - starts with number", "1user", true},
		{"invalid - starts with hyphen", "-user", true},
		{"invalid - empty", "", true},
		{"invalid - too long", "thisusernameiswaytoolongtobevalidandexceedsthirtytwocharacters", true},
		{"invalid - special chars", "user@domain", true},
		{"invalid - space", "john doe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid domain", "example.com", false},
		{"valid subdomain", "sub.example.com", false},
		{"valid localhost", "localhost", false},
		{"valid with hyphen", "my-server.example.com", false},
		{"invalid - empty", "", true},
		{"invalid - starts with hyphen", "-example.com", true},
		{"invalid - ends with hyphen", "example-.com", true},
		{"invalid - double dot", "example..com", true},
		{"invalid - special chars", "example@domain.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name    string
		tz      string
		wantErr bool
	}{
		{"valid timezone", "America/Chicago", false},
		{"valid timezone - Europe", "Europe/London", false},
		{"invalid - no slash", "EST", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimezone(tt.tz)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimezone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
