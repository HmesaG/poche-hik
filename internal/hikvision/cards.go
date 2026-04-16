package hikvision

import (
	"context"
	"encoding/xml"
	"fmt"
)

// CardInfo represents a card record on the device
type CardInfo struct {
	EmployeeNo string `xml:"employeeNo"`
	CardNo     string `xml:"cardNo"`
	CardType   string `xml:"cardType"` // normalCard
}

// CreateCard registers a new card for an employee
func (c *Client) CreateCard(ctx context.Context, card CardInfo) error {
	body, err := xml.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal card: %w", err)
	}

	_, err = c.Do(ctx, "POST", "/ISAPI/AccessControl/CardInfo/Record", nil, body)
	return err
}

// DeleteCard deletes a card record by card number
func (c *Client) DeleteCard(ctx context.Context, cardNo string) error {
	type CardInfoDelCond struct {
		CardNoList []struct {
			CardNo string `xml:"cardNo"`
		} `xml:"CardNoList>CardNo"`
	}

	cond := CardInfoDelCond{
		CardNoList: []struct {
			CardNo string `xml:"cardNo"`
		}{{CardNo: cardNo}},
	}

	body, err := xml.Marshal(cond)
	if err != nil {
		return fmt.Errorf("marshal delete card cond: %w", err)
	}

	_, err = c.Do(ctx, "PUT", "/ISAPI/AccessControl/CardInfo/Delete", nil, body)
	return err
}
