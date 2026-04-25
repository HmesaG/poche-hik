package config

import "strconv"

// ApplyOverrides overlays persisted values on top of the runtime config.
func ApplyOverrides(cfg *Config, values map[string]string) {
	if cfg == nil {
		return
	}

	if value, ok := values["server_port"]; ok && value != "" {
		cfg.ServerPort = value
	}
	if value, ok := values["log_level"]; ok && value != "" {
		cfg.LogLevel = value
	}
	if value, ok := values["company_name"]; ok && value != "" {
		cfg.CompanyName = value
	}
	if value, ok := values["company_rnc"]; ok && value != "" {
		cfg.CompanyRNC = value
	}
	if value, ok := values["hikvision_ip"]; ok && value != "" {
		cfg.HikvisionIP = value
	}
	if value, ok := values["hikvision_port"]; ok && value != "" {
		cfg.HikvisionPort = parseInt(value, cfg.HikvisionPort)
	}
	if value, ok := values["hikvision_username"]; ok && value != "" {
		cfg.HikvisionUsername = value
	}
	if value, ok := values["hikvision_password"]; ok && value != "" {
		cfg.HikvisionPassword = value
	}
	if value, ok := values["default_shift_start"]; ok && value != "" {
		cfg.DefaultShiftStart = value
	}
	if value, ok := values["default_shift_end"]; ok && value != "" {
		cfg.DefaultShiftEnd = value
	}
	if value, ok := values["lunch_break_minutes"]; ok && value != "" {
		cfg.LunchBreakMinutes = parseInt(value, cfg.LunchBreakMinutes)
	}
	if value, ok := values["grace_period_minutes"]; ok && value != "" {
		cfg.GracePeriodMinutes = parseInt(value, cfg.GracePeriodMinutes)
	}
	if value, ok := values["overtime_multiplier_simple"]; ok && value != "" {
		cfg.OvertimeMultiplierSimple = parseFloat(value, cfg.OvertimeMultiplierSimple)
	}
	if value, ok := values["overtime_multiplier_double"]; ok && value != "" {
		cfg.OvertimeMultiplierDouble = parseFloat(value, cfg.OvertimeMultiplierDouble)
	}
	if value, ok := values["overtime_multiplier_triple"]; ok && value != "" {
		cfg.OvertimeMultiplierTriple = parseFloat(value, cfg.OvertimeMultiplierTriple)
	}
	if value, ok := values["overtime_threshold_hours"]; ok && value != "" {
		cfg.OvertimeThresholdHours = parseFloat(value, cfg.OvertimeThresholdHours)
	}
	if value, ok := values["ldap_host"]; ok && value != "" {
		cfg.LDAPHost = value
	}
	if value, ok := values["ldap_port"]; ok && value != "" {
		cfg.LDAPPort = parseInt(value, cfg.LDAPPort)
	}
	if value, ok := values["ldap_base_dn"]; ok && value != "" {
		cfg.LDAPBaseDN = value
	}
	if value, ok := values["ldap_bind_dn"]; ok && value != "" {
		cfg.LDAPBindDN = value
	}
	if value, ok := values["ldap_bind_pass"]; ok && value != "" {
		cfg.LDAPBindPass = value
	}
	if value, ok := values["ldap_user_filter"]; ok && value != "" {
		cfg.LDAPUserFilter = value
	}
	if value, ok := values["ldap_dept_filter"]; ok && value != "" {
		cfg.LDAPDeptFilter = value
	}
	if value, ok := values["ldap_pos_filter"]; ok && value != "" {
		cfg.LDAPPosFilter = value
	}
	if value, ok := values["jwt_secret"]; ok && value != "" {
		cfg.JWTSecret = value
	}
	if value, ok := values["jwt_expiration_hours"]; ok && value != "" {
		cfg.JWTExpiration = parseInt(value, cfg.JWTExpiration)
	}
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
