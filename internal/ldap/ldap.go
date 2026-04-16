package ldap

import (
	"context"
	"fmt"
	"ponches/internal/config"
	"ponches/internal/employees"
	"ponches/internal/store"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Syncer struct {
	cfg   *config.Config
	store store.Repository
}

func NewSyncer(cfg *config.Config, s store.Repository) *Syncer {
	return &Syncer{cfg: cfg, store: s}
}

func (s *Syncer) Connect() (*ldap.Conn, error) {
	address := fmt.Sprintf("%s:%d", s.cfg.LDAPHost, s.cfg.LDAPPort)
	l, err := ldap.DialURL("ldap://" + address)
	if err != nil {
		return nil, fmt.Errorf("ldap dial: %w", err)
	}

	if s.cfg.LDAPBindDN != "" {
		err = l.Bind(s.cfg.LDAPBindDN, s.cfg.LDAPBindPass)
		if err != nil {
			l.Close()
			return nil, fmt.Errorf("ldap bind: %w", err)
		}
	}

	return l, nil
}

func (s *Syncer) Sync(ctx context.Context) error {
	l, err := s.Connect()
	if err != nil {
		return err
	}
	defer l.Close()

	log.Info().Msg("LDAP Sync: Starting...")

	// 1. Sync Departments (and positions if possible)
	// For simplicity, we'll extract them from users if they don't have a specific OU sync
	
	// 2. Sync Users
	searchRequest := ldap.NewSearchRequest(
		s.cfg.LDAPBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		s.cfg.LDAPUserFilter,
		[]string{"sAMAccountName", "employeeNumber", "givenName", "sn", "mail", "department", "title", "distinguishedName"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("ldap search: %w", err)
	}

	for _, entry := range sr.Entries {
		username := entry.GetAttributeValue("sAMAccountName")
		empNo := entry.GetAttributeValue("employeeNumber")
		if empNo == "" {
			empNo = username // Fallback
		}
		if empNo == "" {
			continue
		}

		firstName := entry.GetAttributeValue("givenName")
		lastName := entry.GetAttributeValue("sn")
		email := entry.GetAttributeValue("mail")
		deptName := entry.GetAttributeValue("department")
		posName := entry.GetAttributeValue("title")

		// Ensure Department exists
		var deptID string
		if deptName != "" {
			deptID = strings.ToLower(deptName)
			err = s.store.UpsertDepartment(ctx, &employees.Department{
				ID:   deptID,
				Name: deptName,
			})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to upsert department: %s", deptName)
			}
		}

		// Ensure Position exists
		var posID string
		if posName != "" {
			posID = strings.ToLower(posName)
			err = s.store.UpsertPosition(ctx, &employees.Position{
				ID:           posID,
				Name:         posName,
				DepartmentID: deptID,
			})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to upsert position: %s", posName)
			}
		}

		// Upsert Employee
		// We'll use sAMAccountName as ID if UUID is not provided, or better, use stable ID
		// For now, let's use a hash of DN or just sAMAccountName if it's unique
		emp := &employees.Employee{
			ID:           entry.GetAttributeValue("distinguishedName"), // Using DN as ID for stability
			EmployeeNo:   empNo,
			FirstName:    firstName,
			LastName:     lastName,
			Email:        email,
			DepartmentID: deptID,
			PositionID:   posID,
			Status:       "Active",
		}
		
		if emp.ID == "" {
			emp.ID = uuid.New().String()
		}

		err = s.store.UpsertEmployee(ctx, emp)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to upsert employee: %s", empNo)
		}
	}

	log.Info().Msgf("LDAP Sync: Completed. Processed %d entries", len(sr.Entries))
	return nil
}
