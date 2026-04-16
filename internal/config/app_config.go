package config

// AppConfig represents the application configuration stored in the database
type AppConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Configuration keys
const (
	// Server
	ConfigKeyServerPort     = "server_port"
	ConfigKeyLogLevel       = "log_level"
	
	// Hikvision Device
	ConfigKeyHikvisionIP       = "hikvision_ip"
	ConfigKeyHikvisionUsername = "hikvision_username"
	ConfigKeyHikvisionPassword = "hikvision_password"
	
	// Attendance Rules
	ConfigKeyDefaultShiftStart  = "default_shift_start"
	ConfigKeyDefaultShiftEnd    = "default_shift_end"
	ConfigKeyLunchBreakMinutes  = "lunch_break_minutes"
	ConfigKeyGracePeriodMinutes = "grace_period_minutes"
	
	// Overtime
	ConfigKeyOvertimeMultiplierSimple  = "overtime_multiplier_simple"
	ConfigKeyOvertimeMultiplierDouble  = "overtime_multiplier_double"
	ConfigKeyOvertimeMultiplierTriple  = "overtime_multiplier_triple"
	ConfigKeyOvertimeThresholdHours    = "overtime_threshold_hours"
	
	// LDAP
	ConfigKeyLDAPHost       = "ldap_host"
	ConfigKeyLDAPPort       = "ldap_port"
	ConfigKeyLDAPBaseDN     = "ldap_base_dn"
	ConfigKeyLDAPBindDN     = "ldap_bind_dn"
	ConfigKeyLDAPBindPass   = "ldap_bind_pass"
	ConfigKeyLDAPUserFilter = "ldap_user_filter"
	ConfigKeyLDAPDeptFilter = "ldap_dept_filter"
	ConfigKeyLDAPPosFilter  = "ldap_pos_filter"
	
	// JWT
	ConfigKeyJWTSecret     = "jwt_secret"
	ConfigKeyJWTExpiration = "jwt_expiration_hours"

	// Company Info
	ConfigKeyCompanyName = "company_name"
	ConfigKeyCompanyRNC  = "company_rnc"
)

// DefaultConfig returns default values for all configuration keys
func DefaultConfig() map[string]string {
	return map[string]string{
		ConfigKeyServerPort:              "8080",
		ConfigKeyLogLevel:                "info",
		ConfigKeyHikvisionIP:             "192.168.1.64",
		ConfigKeyHikvisionUsername:       "admin",
		ConfigKeyHikvisionPassword:       "",
		ConfigKeyDefaultShiftStart:       "08:00",
		ConfigKeyDefaultShiftEnd:         "17:00",
		ConfigKeyLunchBreakMinutes:       "60",
		ConfigKeyGracePeriodMinutes:      "5",
		ConfigKeyOvertimeMultiplierSimple: "1.5",
		ConfigKeyOvertimeMultiplierDouble: "2.0",
		ConfigKeyOvertimeMultiplierTriple: "3.0",
		ConfigKeyOvertimeThresholdHours:  "8.0",
		ConfigKeyLDAPHost:                "",
		ConfigKeyLDAPPort:                "389",
		ConfigKeyLDAPBaseDN:              "",
		ConfigKeyLDAPBindDN:              "",
		ConfigKeyLDAPBindPass:            "",
		ConfigKeyLDAPUserFilter:          "(objectClass=person)",
		ConfigKeyLDAPDeptFilter:          "(objectClass=organizationalUnit)",
		ConfigKeyLDAPPosFilter:           "(objectClass=group)",
		ConfigKeyJWTSecret:               "change-this-secret-in-production",
		ConfigKeyJWTExpiration:           "24",
		ConfigKeyCompanyName:             "Empresa",
		ConfigKeyCompanyRNC:              "",
	}
}
