package hikvision

import (
	"context"
	"encoding/xml"
	"fmt"
)

// UserRecord represents a user in the access control device
type UserRecord struct {
	EmployeeNo string `xml:"employeeNo"`
	Name       string `xml:"name"`
	UserType   string `xml:"userType"` // normal, visitor, etc.
	Valid      struct {
		Enable bool   `xml:"enable"`
		Begin  string `xml:"beginTime"`
		End    string `xml:"endTime"`
	} `xml:"Valid"`
	DoorRight string `xml:"doorRight"`
	RightPlan []struct {
		DoorNo   int    `xml:"doorNo"`
		PlanNo   string `xml:"planTemplateNo"`
	} `xml:"RightPlan"`
}

// UserInfoSearchCond represents the condition for searching users
type UserInfoSearchCond struct {
	XMLName    xml.Name `xml:"UserInfoSearchCond"`
	SearchID   string   `xml:"searchID"`
	MaxResults int      `xml:"maxResults"`
	SearchResultOffset int `xml:"searchResultOffset"`
}

// UserInfoSearchResult represents the search results
type UserInfoSearchResult struct {
	UserInfoList []UserRecord `xml:"UserInfo"`
	TotalMatches int          `xml:"totalMatches"`
	IsSearchDone bool         `xml:"isSearchDone"`
}

// CreateUser creates a new user on the device
func (c *Client) CreateUser(ctx context.Context, user UserRecord) error {
	body, err := xml.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	_, err = c.Do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/Record", nil, body)
	return err
}

// SearchUsers searches for users on the device
func (c *Client) SearchUsers(ctx context.Context, offset, limit int) (*UserInfoSearchResult, error) {
	cond := UserInfoSearchCond{
		SearchID:           "search_users",
		MaxResults:         limit,
		SearchResultOffset: offset,
	}

	body, err := xml.Marshal(cond)
	if err != nil {
		return nil, fmt.Errorf("marshal search cond: %w", err)
	}

	resp, err := c.Do(ctx, "POST", "/ISAPI/AccessControl/UserInfo/Search", nil, body)
	if err != nil {
		return nil, err
	}

	var result UserInfoSearchResult
	if err := xml.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal search result: %w", err)
	}

	return &result, nil
}

// DeleteUser deletes a user by their employeeNo
func (c *Client) DeleteUser(ctx context.Context, employeeNo string) error {
	type UserInfoDelCond struct {
		EmployeeNoList []struct {
			EmployeeNo string `xml:"employeeNo"`
		} `xml:"EmployeeNoList>EmployeeNo"`
	}

	cond := UserInfoDelCond{
		EmployeeNoList: []struct {
			EmployeeNo string `xml:"employeeNo"`
		}{{EmployeeNo: employeeNo}},
	}

	body, err := xml.Marshal(cond)
	if err != nil {
		return fmt.Errorf("marshal delete cond: %w", err)
	}

	_, err = c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Delete", nil, body)
	return err
}
