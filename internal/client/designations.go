package client

import (
	"fmt"
	"net/url"
	"strconv"
)

// DesignateContact assigns a contact to a box.
func (c *Client) DesignateContact(boxID int64, contactID int64) error {
	values := url.Values{
		"contact_id": {strconv.FormatInt(contactID, 10)},
	}
	_, err := c.PostForm(fmt.Sprintf("/boxes/%d/designations", boxID), values)
	return err
}
