package client

import (
	"fmt"
	"net/url"
)

// CreateEvent creates a calendar event and returns the redirect response with the new event ID.
func (c *Client) CreateEvent(values url.Values) (*RedirectResponse, error) {
	return c.PostForm("/calendar/events", values)
}

// UpdateEvent updates a calendar event by ID.
func (c *Client) UpdateEvent(id int64, values url.Values) error {
	return c.PatchForm(fmt.Sprintf("/calendar/events/%d", id), values)
}

// DeleteEvent deletes a calendar event by ID.
func (c *Client) DeleteEvent(id int64) error {
	return c.Delete(fmt.Sprintf("/calendar/events/%d", id))
}
