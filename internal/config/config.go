package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	ServerPort string
	Port       int
	ServerIP   string
	DBPath     string
	LogLevel   string

	// Company Info
	CompanyName string
	CompanyRNC  string

	// Scanner
	SADPTimeoutSeconds int

	// Default Device Credentials
	HikvisionIP       string
	HikvisionPort     int
	HikvisionUsername string
	HikvisionPassword string

	// Attendance Rules
	DefaultShiftStart  string
	DefaultShiftEnd    string
	WeeklyScheduleJSON string
	LunchBreakMinutes  int
	GracePeriodMinutes int

	// Overtime Mutlipliers
	OvertimeMultiplierSimple float64
	OvertimeMultiplierDouble float64
	OvertimeMultiplierTriple float64
	OvertimeThresholdHours   float64

	// LDAP Configuration
	LDAPHost        string
	LDAPPort        int
	LDAPBaseDN      string
	LDAPBindDN      string
	LDAPBindPass    string
	LDAPUserFilter  string
	LDAPDeptFilter  string
	LDAPPosFilter   string

	// JWT Authentication
	JWTSecret     string
	JWTExpiration int // in hours

	// Modular Features
	TravelEnabled bool
}

// Load loads the configuration from environment variables or .env file
func Load() (*Config, error) {
	// Attempt to load .env if it exists
	if err := godotenv.Load(); err != nil {
		log.Debug().Msg(".env file not found, relying on environment variables only")
	}

	cfg := &Config{
		ServerPort:               getEnv("SERVER_PORT", "8080"),
		Port:                     getEnvAsInt("SERVER_PORT", 8080),
		ServerIP:                 getEnv("SERVER_IP", "127.0.0.1"),
		DBPath:                   getEnv("DB_PATH", filepath.Join(".", "ponches.db")),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		CompanyName:              getEnv("COMPANY_NAME", "Empresa"),
		CompanyRNC:               getEnv("COMPANY_RNC", ""),
		SADPTimeoutSeconds:       getEnvAsInt("SADP_TIMEOUT_SECONDS", 5),
		HikvisionIP:              getEnv("HIKVISION_IP", "192.168.1.64"),
		HikvisionPort:            getEnvAsInt("HIKVISION_PORT", 80),
		HikvisionUsername:        getEnv("HIKVISION_USERNAME", ""),
		HikvisionPassword:        getEnv("HIKVISION_PASSWORD", ""),
		DefaultShiftStart:        getEnv("DEFAULT_SHIFT_START", "08:00"),
		DefaultShiftEnd:          getEnv("DEFAULT_SHIFT_END", "17:00"),
		WeeklyScheduleJSON:       getEnv("WEEKLY_SCHEDULE", ""),
		LunchBreakMinutes:        getEnvAsInt("LUNCH_BREAK_MINUTES", 60),
		GracePeriodMinutes:       getEnvAsInt("GRACE_PERIOD_MINUTES", 5),
		OvertimeMultiplierSimple: getEnvAsFloat("OVERTIME_MULTIPLIER_SIMPLE", 1.5),
		OvertimeMultiplierDouble: getEnvAsFloat("OVERTIME_MULTIPLIER_DOUBLE", 2.0),
		OvertimeMultiplierTriple: getEnvAsFloat("OVERTIME_MULTIPLIER_TRIPLE", 3.0),
		OvertimeThresholdHours:   getEnvAsFloat("OVERTIME_THRESHOLD_HOURS", 8.0),
		LDAPHost:                 getEnv("LDAP_HOST", ""),
		LDAPPort:                 getEnvAsInt("LDAP_PORT", 389),
		LDAPBaseDN:               getEnv("LDAP_BASE_DN", ""),
		LDAPBindDN:               getEnv("LDAP_BIND_DN", ""),
		LDAPBindPass:             getEnv("LDAP_BIND_PASS", ""),
		LDAPUserFilter:           getEnv("LDAP_USER_FILTER", "(objectClass=person)"),
		LDAPDeptFilter:           getEnv("LDAP_DEPT_FILTER", "(objectClass=organizationalUnit)"),
		LDAPPosFilter:            getEnv("LDAP_POS_FILTER", "(objectClass=group)"),
		JWTSecret:                getEnv("JWT_SECRET", "ponches-secret-key-change-in-production"),
		JWTExpiration:            getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		TravelEnabled:            getEnvAsBool("TRAVEL_ENABLED", true),
	}

	return cfg, nil
}

// Helper functions
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultVal
}
