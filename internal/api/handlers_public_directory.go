package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"ponches/internal/employees"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

type publicDirectoryEmployee struct {
	ID             string `json:"id"`
	EmployeeNo     string `json:"employeeNo"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	FullName       string `json:"fullName"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	DepartmentID   string `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
	PositionID     string `json:"positionId"`
	PositionName   string `json:"positionName"`
	Status         string `json:"status"`
}

func (s *Server) handlePublicDirectory(w http.ResponseWriter, r *http.Request) {
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list employees")
		return
	}

	departments, err := s.Store.ListDepartments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list departments")
		return
	}

	positions, err := s.Store.ListPositions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list positions")
		return
	}

	departmentNames := make(map[string]string, len(departments))
	for _, department := range departments {
		departmentNames[department.ID] = department.Name
	}

	positionNames := make(map[string]string, len(positions))
	for _, position := range positions {
		positionNames[position.ID] = position.Name
	}

	response := make([]publicDirectoryEmployee, 0, len(emps))
	for _, emp := range emps {
		response = append(response, buildPublicDirectoryEmployee(emp, departmentNames, positionNames))
	}

	sort.Slice(response, func(i, j int) bool {
		return strings.ToLower(response[i].FullName) < strings.ToLower(response[j].FullName)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"employees": response,
		"total":     len(response),
	})
}

func (s *Server) handlePublicDirectoryContact(w http.ResponseWriter, r *http.Request) {
	employeeNo := strings.TrimSpace(chi.URLParam(r, "employeeNo"))
	if employeeNo == "" {
		writeError(w, http.StatusBadRequest, "Employee number is required")
		return
	}

	emp, err := s.Store.GetEmployeeByNo(r.Context(), employeeNo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get employee")
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	departmentName := ""
	if emp.DepartmentID != "" {
		department, err := s.Store.GetDepartment(r.Context(), emp.DepartmentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get department")
			return
		}
		if department != nil {
			departmentName = department.Name
		}
	}

	positionName := ""
	if emp.PositionID != "" {
		position, err := s.Store.GetPosition(r.Context(), emp.PositionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get position")
			return
		}
		if position != nil {
			positionName = position.Name
		}
	}

	fullName := formatEmployeeFullName(emp)
	fileName := url.PathEscape(strings.ReplaceAll(fullName, " ", "_")) + ".vcf"

	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(buildVCard(emp, fullName, departmentName, positionName)))
}

func buildPublicDirectoryEmployee(emp *employees.Employee, departments, positions map[string]string) publicDirectoryEmployee {
	return publicDirectoryEmployee{
		ID:             emp.ID,
		EmployeeNo:     emp.EmployeeNo,
		FirstName:      emp.FirstName,
		LastName:       emp.LastName,
		FullName:       formatEmployeeFullName(emp),
		Phone:          strings.TrimSpace(emp.Phone),
		Email:          strings.TrimSpace(emp.Email),
		DepartmentID:   emp.DepartmentID,
		DepartmentName: departments[emp.DepartmentID],
		PositionID:     emp.PositionID,
		PositionName:   positions[emp.PositionID],
		Status:         emp.Status,
	}
}

func formatEmployeeFullName(emp *employees.Employee) string {
	return strings.TrimSpace(strings.TrimSpace(emp.FirstName) + " " + strings.TrimSpace(emp.LastName))
}

func buildVCard(emp *employees.Employee, fullName, departmentName, positionName string) string {
	lines := []string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"N:" + vCardEscape(strings.TrimSpace(emp.LastName)) + ";" + vCardEscape(strings.TrimSpace(emp.FirstName)) + ";;;",
		"FN:" + vCardEscape(fullName),
	}

	orgParts := []string{"Grupo MV"}
	if departmentName != "" {
		orgParts = append(orgParts, departmentName)
	}
	lines = append(lines, "ORG:"+vCardEscape(strings.Join(orgParts, ";")))

	if positionName != "" {
		lines = append(lines, "TITLE:"+vCardEscape(positionName))
	}
	if phone := normalizePhoneForVCard(emp.Phone); phone != "" {
		lines = append(lines, "TEL;TYPE=CELL:"+vCardEscape(phone))
	}
	if email := strings.TrimSpace(emp.Email); email != "" {
		lines = append(lines, "EMAIL;TYPE=INTERNET:"+vCardEscape(email))
	}

	if len(emp.PhotoData) > 0 {
		// Encode photo to base64
		photoBase64 := base64.StdEncoding.EncodeToString(emp.PhotoData)
		lines = append(lines, "PHOTO;ENCODING=b;TYPE=JPEG:"+photoBase64)
	}

	publicURL := "/directorio"
	if emp.EmployeeNo != "" {
		publicURL += "#empleado-" + url.PathEscape(emp.EmployeeNo)
	}
	lines = append(lines,
		"NOTE:"+vCardEscape("Contacto exportado desde el directorio interno de la organizacion."),
		"URL:"+vCardEscape(publicURL),
		"END:VCARD",
	)

	return strings.Join(lines, "\r\n") + "\r\n"
}

func vCardEscape(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		";", `\;`,
		",", `\,`,
		"\n", `\n`,
		"\r", "",
	)
	return replacer.Replace(strings.TrimSpace(value))
}

func normalizePhoneForVCard(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	var builder strings.Builder
	for i, r := range phone {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
			continue
		}
		if r == '+' && i == 0 {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
