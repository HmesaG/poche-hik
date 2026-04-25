package hikvision

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"ponches/internal/employees"
)

// --------------------------------------------------------------------------
// XML structures — K1T343EWX / ISAPI AccessControl
// --------------------------------------------------------------------------

// userInfoList wraps UserInfo records for bulk creation/update.
// The device expects: POST /ISAPI/AccessControl/UserInfo/Record
type userInfoList struct {
	XMLName  xml.Name      `xml:"UserInfoList" json:"-"`
	XMLNS    string        `xml:"xmlns,attr" json:"-"`
	UserInfo []userInfoXML `xml:"UserInfo" json:"UserInfo"`
}

// userInfoXML is the full ISAPI representation of a user/employee on the device.
type userInfoXML struct {
	EmployeeNo string `xml:"employeeNo" json:"employeeNo"`
	Name       string `xml:"name" json:"name"`
	// userType: normal | visitor | blackList | administrator
	UserType string `xml:"userType" json:"userType"`
	// gender: male | female
	Gender     string         `xml:"gender,omitempty" json:"gender,omitempty"`
	Valid      validXML       `xml:"Valid" json:"Valid"`
	DoorRight  string         `xml:"doorRight" json:"doorRight"`
	RightPlan  []rightPlanXML `xml:"RightPlan" json:"RightPlan"`
}

type validXML struct {
	Enable    bool   `xml:"enable" json:"enable"`
	BeginTime string `xml:"beginTime" json:"beginTime"`
	EndTime   string `xml:"endTime" json:"endTime"`
}

type rightPlanXML struct {
	DoorNo         int    `xml:"doorNo" json:"doorNo"`
	PlanTemplateNo string `xml:"planTemplateNo" json:"planTemplateNo"`
}

// userInfoSearchCond is the search request body for /ISAPI/AccessControl/UserInfo/Search.
type userInfoSearchCond struct {
	XMLName            xml.Name `xml:"UserInfoSearchCond"`
	XMLNS              string   `xml:"xmlns,attr"`
	SearchID           string   `xml:"searchID"`
	MaxResults         int      `xml:"maxResults"`
	SearchResultOffset int      `xml:"searchResultOffset"`
}

// userInfoSearchResult is the response from the search endpoint.
type userInfoSearchResult struct {
	XMLName      xml.Name      `xml:"UserInfoSearch"`
	ResponseStatusCode int     `xml:"responseStatusCode"`
	TotalMatches int           `xml:"totalMatches"`
	IsSearchDone bool          `xml:"isSearchDone"`
	UserInfoList []userInfoXML `xml:"UserInfoList>UserInfo"`
}

// userInfoDelCond is the request body for /ISAPI/AccessControl/UserInfo/Delete.
type userInfoDelCond struct {
	XMLName     xml.Name `xml:"UserInfoDelCond"`
	XMLNS       string   `xml:"xmlns,attr"`
	EmployeeIDs []employeeNoXML `xml:"EmployeeNoList>EmployeeNo"`
}

type employeeNoXML struct {
	EmployeeNo string `xml:"employeeNo"`
}

const (
	isapiNS         = "http://www.isapi.org/ver20/XMLSchema"
	defaultBeginTime = "2020-01-01T00:00:00"
	defaultEndTime   = "2037-12-31T23:59:59"
	defaultDoorRight = "1"
	defaultPlanNo    = "1"
)

// xmlHeader prepends the standard XML declaration expected by K1T343EWX.
func xmlHeader(payload []byte) []byte {
	return append([]byte(xml.Header), payload...)
}

// --------------------------------------------------------------------------
// Adapter: employees.Employee ↔ userInfoXML
// --------------------------------------------------------------------------

// EmployeeToUserInfo maps an internal Employee to the Hikvision ISAPI user struct.
//
// Mapping rules:
//   - EmployeeNo      → employeeNo (1:1)
//   - FirstName+" "+LastName → name (max 32 chars per ISAPI spec)
//   - Status == "Active" → Valid.Enable = true; dates are set to open-ended defaults
//   - Gender           → gender (male|female); anything else is omitted
//   - Access is granted to door 1 by default with unrestricted schedule
func EmployeeToUserInfo(e *employees.Employee) *userInfoXML {
	name := strings.TrimSpace(e.FirstName + " " + e.LastName)
	if len(name) > 32 {
		name = name[:32]
	}

	enable := strings.EqualFold(e.Status, "active")
	gender := normalizeGender(e.Gender)

	return &userInfoXML{
		EmployeeNo: e.EmployeeNo,
		Name:       name,
		UserType:   "normal",
		Gender:     gender,
		Valid: validXML{
			Enable:    enable,
			BeginTime: defaultBeginTime,
			EndTime:   defaultEndTime,
		},
		DoorRight: defaultDoorRight,
		RightPlan: []rightPlanXML{
			{DoorNo: 1, PlanTemplateNo: defaultPlanNo},
		},
	}
}

// normalizeGender maps free-text gender values to ISAPI accepted values.
func normalizeGender(g string) string {
	switch strings.ToLower(g) {
	case "male", "masculino", "m":
		return "male"
	case "female", "femenino", "f":
		return "female"
	default:
		return "" // omitempty — field will be skipped
	}
}

// UserInfoToEmployee performs a partial reverse mapping — useful when syncing
// device users back to the internal store without overwriting all fields.
func UserInfoToEmployee(u *userInfoXML) *employees.Employee {
	parts := strings.SplitN(u.Name, " ", 2)
	first, last := parts[0], ""
	if len(parts) > 1 {
		last = parts[1]
	}

	status := "Inactive"
	if u.Valid.Enable {
		status = "Active"
	}

	return &employees.Employee{
		EmployeeNo: u.EmployeeNo,
		FirstName:  first,
		LastName:   last,
		Gender:     u.Gender,
		Status:     status,
	}
}

// --------------------------------------------------------------------------
// CRUD operations — UserInfo
// --------------------------------------------------------------------------

// CreateUser registers a new user on the Hikvision device.
func (c *Client) CreateUser(ctx context.Context, emp *employees.Employee) error {
	return c.upsertBatch(ctx, []*employees.Employee{emp})
}

// UpdateUser updates an existing user on the device.
// The ISAPI uses PUT to the same endpoint for updates.
func (c *Client) UpdateUser(ctx context.Context, emp *employees.Employee) error {
	info := EmployeeToUserInfo(emp)
	list := userInfoList{
		XMLNS:    isapiNS,
		UserInfo: []userInfoXML{*info},
	}

	payload, err := xml.Marshal(list)
	if err != nil {
		return fmt.Errorf("marshal user update: %w", err)
	}

	if _, err := c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Record", nil, xmlHeader(payload)); err != nil {
		return fmt.Errorf("update user %q: %w", emp.EmployeeNo, err)
	}
	return nil
}

// CreateUsers registers multiple employees on the device one by one.
// The K1T343EWX /Record endpoint accepts a single UserInfo object per request.
func (c *Client) CreateUsers(ctx context.Context, emps []*employees.Employee) error {
	for _, emp := range emps {
		if err := c.upsertBatch(ctx, []*employees.Employee{emp}); err != nil {
			return fmt.Errorf("create user %q: %w", emp.EmployeeNo, err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// CRUD operations — User Search / Delete
// --------------------------------------------------------------------------

// GetUsers retrieves all users from the device, paginating automatically.
// Returns a slice of raw ISAPI user structs; use UserInfoToEmployee for conversion.
func (c *Client) GetUsers(ctx context.Context) ([]userInfoXML, error) {
	const pageSize = 20
	var all []userInfoXML
	offset := 0

	for {
		cond := userInfoSearchCond{
			XMLNS:              isapiNS,
			SearchID:           fmt.Sprintf("get_users_%d_%d", offset, time.Now().UnixMilli()),
			MaxResults:         pageSize,
			SearchResultOffset: offset,
		}

		payload, err := xml.Marshal(cond)
		if err != nil {
			return nil, fmt.Errorf("marshal search cond: %w", err)
		}

		resp, err := c.Do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/Search", nil, xmlHeader(payload))
		if err != nil {
			return nil, fmt.Errorf("search users (offset=%d): %w", offset, err)
		}

		var result userInfoSearchResult
		if err := xml.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("unmarshal users response: %w (body: %s)", err, string(resp))
		}

		all = append(all, result.UserInfoList...)

		if result.IsSearchDone || len(all) >= result.TotalMatches {
			break
		}
		offset += pageSize
	}

	return all, nil
}

// DeleteUser removes a user from the device by employeeNo.
func (c *Client) DeleteUser(ctx context.Context, employeeNo string) error {
	// K1T343EWX expects JSON format for delete operations
	// Standard ISAPI JSON structure for deletion
	req := map[string]interface{}{
		"UserInfoDelCond": map[string]interface{}{
			"EmployeeNoList": []map[string]interface{}{
				{"employeeNo": employeeNo},
			},
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal delete cond: %w", err)
	}

	if _, err := c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Delete?format=json",
		map[string]string{"Content-Type": "application/json"}, payload); err != nil {
		return fmt.Errorf("delete user %q: %w", employeeNo, err)
	}
	return nil
}

// DeleteUsers removes multiple users in one ISAPI call.
// The K1T343EWX accepts up to 100 employeeNo entries per request.
func (c *Client) DeleteUsers(ctx context.Context, employeeNos []string) error {
	ids := make([]employeeNoXML, len(employeeNos))
	for i, no := range employeeNos {
		ids[i] = employeeNoXML{EmployeeNo: no}
	}

	cond := userInfoDelCond{
		XMLNS:       isapiNS,
		EmployeeIDs: ids,
	}

	payload, err := xml.Marshal(cond)
	if err != nil {
		return fmt.Errorf("marshal batch delete cond: %w", err)
	}

	if _, err := c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Delete", nil, xmlHeader(payload)); err != nil {
		return fmt.Errorf("batch delete users: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------
// Private helpers
// --------------------------------------------------------------------------

// upsertBatch sends a single employee to the device as a JSON object.
// If the user already exists (POST fails with employeeNoAlreadyExist), it deletes and recreates.
func (c *Client) upsertBatch(ctx context.Context, emps []*employees.Employee) error {
	for _, emp := range emps {
		info := EmployeeToUserInfo(emp)

		// Wrap in a single-user envelope matching what K1T343EWX accepts
		envelope := map[string]interface{}{
			"UserInfo": info,
		}

		payload, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("marshal user %q: %w", emp.EmployeeNo, err)
		}

		// Try POST (Create)
		_, err = c.Do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/Record?format=json",
			map[string]string{"Content-Type": "application/json"}, payload)

		if err != nil {
			// Check if it's a "already exists" error
			if isAPIErr, ok := err.(*ISAPIError); ok && isAPIErr.StatusCode == 400 {
				bodyStr := string(isAPIErr.Body)
				if strings.Contains(bodyStr, "employeeNoAlreadyExist") || strings.Contains(bodyStr, "Duplicate") {
					// User exists - Try Update via Modify endpoint first (more efficient)
					_, modErr := c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Modify?format=json",
						map[string]string{"Content-Type": "application/json"}, payload)
					
					if modErr == nil {
						continue
					}

					// If Modify fails, fall back to Delete + Recreate (nuclear option)
					log.Debug().Str("employeeNo", emp.EmployeeNo).Msg("Modify failed, attempting delete and recreate")
					if delErr := c.DeleteUser(ctx, emp.EmployeeNo); delErr != nil {
						return fmt.Errorf("delete existing user %q before update: %w", emp.EmployeeNo, delErr)
					}
					// Retry POST after delete
					if _, postErr := c.Do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/Record?format=json",
						map[string]string{"Content-Type": "application/json"}, payload); postErr != nil {
						return fmt.Errorf("recreate user %q after delete: %w", emp.EmployeeNo, postErr)
					}
					continue
				}
			}
			return fmt.Errorf("create user %q: %w", emp.EmployeeNo, err)
		}
	}
	return nil
}
