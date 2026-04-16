package api

import (
	"encoding/json"
	"net/http"
	"ponches/internal/auth"
	"ponches/internal/ldap"
	"strconv"
)

// handleGetConfig returns the current configuration
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get config from database
	dbConfig, err := s.Store.GetAllConfig(r.Context())
	if err != nil {
		dbConfig = make(map[string]string)
	}

	// Merge with current config for display
	response := map[string]string{
		"server_port":                s.Config.ServerPort,
		"log_level":                  s.Config.LogLevel,
		"company_name":               s.Config.CompanyName,
		"company_rnc":                s.Config.CompanyRNC,
		"hikvision_ip":               s.Config.HikvisionIP,
		"hikvision_username":         s.Config.HikvisionUsername,
		"default_shift_start":        s.Config.DefaultShiftStart,
		"default_shift_end":          s.Config.DefaultShiftEnd,
		"lunch_break_minutes":        strconv.Itoa(s.Config.LunchBreakMinutes),
		"grace_period_minutes":       strconv.Itoa(s.Config.GracePeriodMinutes),
		"overtime_multiplier_simple": strconv.FormatFloat(s.Config.OvertimeMultiplierSimple, 'f', 2, 64),
		"overtime_multiplier_double": strconv.FormatFloat(s.Config.OvertimeMultiplierDouble, 'f', 2, 64),
		"overtime_multiplier_triple": strconv.FormatFloat(s.Config.OvertimeMultiplierTriple, 'f', 2, 64),
		"overtime_threshold_hours":   strconv.FormatFloat(s.Config.OvertimeThresholdHours, 'f', 2, 64),
		"ldap_host":                  s.Config.LDAPHost,
		"ldap_port":                  strconv.Itoa(s.Config.LDAPPort),
		"ldap_base_dn":               s.Config.LDAPBaseDN,
		"ldap_bind_dn":               s.Config.LDAPBindDN,
		"ldap_user_filter":           s.Config.LDAPUserFilter,
		"jwt_expiration_hours":       strconv.Itoa(s.Config.JWTExpiration),
	}

	// Override with DB values if they exist
	for key, value := range dbConfig {
		if value != "" {
			response[key] = value
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// handleUpdateConfig updates the configuration
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg map[string]string
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if val, ok := newCfg["device_ip"]; ok {
		newCfg["hikvision_ip"] = val
		delete(newCfg, "device_ip")
	}
	if val, ok := newCfg["device_user"]; ok {
		newCfg["hikvision_username"] = val
		delete(newCfg, "device_user")
	}
	if val, ok := newCfg["device_pass"]; ok {
		newCfg["hikvision_password"] = val
		delete(newCfg, "device_pass")
	}
	if val, ok := newCfg["grace_period"]; ok {
		newCfg["grace_period_minutes"] = val
		delete(newCfg, "grace_period")
	}
	if val, ok := newCfg["work_hours"]; ok {
		newCfg["overtime_threshold_hours"] = val
		delete(newCfg, "work_hours")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := r.Context()

	// Update company info
	if val, ok := newCfg["company_name"]; ok {
		s.Config.CompanyName = val
		s.Store.SetConfigValue(ctx, "company_name", val)
	}
	if val, ok := newCfg["company_rnc"]; ok {
		s.Config.CompanyRNC = val
		s.Store.SetConfigValue(ctx, "company_rnc", val)
	}

	// Update in-memory config
	if val, ok := newCfg["hikvision_ip"]; ok {
		s.Config.HikvisionIP = val
	}
	if val, ok := newCfg["hikvision_username"]; ok {
		s.Config.HikvisionUsername = val
	}
	if val, ok := newCfg["hikvision_password"]; ok && val != "" {
		s.Config.HikvisionPassword = val
	}

	if val, ok := newCfg["default_shift_start"]; ok {
		s.Config.DefaultShiftStart = val
	}
	if val, ok := newCfg["default_shift_end"]; ok {
		s.Config.DefaultShiftEnd = val
	}
	if val, ok := newCfg["lunch_break_minutes"]; ok {
		if v, err := strconv.Atoi(val); err == nil {
			s.Config.LunchBreakMinutes = v
		}
	}
	if val, ok := newCfg["grace_period_minutes"]; ok {
		if v, err := strconv.Atoi(val); err == nil {
			s.Config.GracePeriodMinutes = v
		}
	}

	if val, ok := newCfg["overtime_multiplier_simple"]; ok {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.Config.OvertimeMultiplierSimple = v
		}
	}
	if val, ok := newCfg["overtime_multiplier_double"]; ok {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.Config.OvertimeMultiplierDouble = v
		}
	}
	if val, ok := newCfg["overtime_multiplier_triple"]; ok {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.Config.OvertimeMultiplierTriple = v
		}
	}
	if val, ok := newCfg["overtime_threshold_hours"]; ok {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			s.Config.OvertimeThresholdHours = v
		}
	}

	// LDAP Config
	if val, ok := newCfg["ldap_host"]; ok {
		s.Config.LDAPHost = val
	}
	if val, ok := newCfg["ldap_port"]; ok {
		if v, err := strconv.Atoi(val); err == nil {
			s.Config.LDAPPort = v
		}
	}
	if val, ok := newCfg["ldap_base_dn"]; ok {
		s.Config.LDAPBaseDN = val
	}
	if val, ok := newCfg["ldap_bind_dn"]; ok {
		s.Config.LDAPBindDN = val
	}
	if val, ok := newCfg["ldap_bind_pass"]; ok && val != "" {
		s.Config.LDAPBindPass = val
	}
	if val, ok := newCfg["ldap_user_filter"]; ok {
		s.Config.LDAPUserFilter = val
	}
	if val, ok := newCfg["ldap_dept_filter"]; ok {
		s.Config.LDAPDeptFilter = val
	}
	if val, ok := newCfg["ldap_pos_filter"]; ok {
		s.Config.LDAPPosFilter = val
	}

	// JWT Config
	if val, ok := newCfg["jwt_secret"]; ok && val != "" {
		s.Config.JWTSecret = val
	}
	if val, ok := newCfg["jwt_expiration_hours"]; ok {
		if v, err := strconv.Atoi(val); err == nil {
			s.Config.JWTExpiration = v
		}
	}

	// Persist to database
	if err := s.Store.SetMultipleConfigValues(ctx, newCfg); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save configuration")
		return
	}

	s.JWTService = auth.NewJWTService(s.Config.JWTSecret, s.Config.JWTExpiration)

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "message": "Configuration saved successfully"})
}

func (s *Server) handleTestLDAP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	syncer := ldap.NewSyncer(s.Config, s.Store)
	l, err := syncer.Connect()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer l.Close()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Conexión exitosa"})
}

func (s *Server) handleSyncLDAP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	syncer := ldap.NewSyncer(s.Config, s.Store)
	err := syncer.Sync(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Sincronización completada"})
}
