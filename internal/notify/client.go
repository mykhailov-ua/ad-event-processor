package notify

type Client struct {
	closeFn func()
	api     NotifierAPI
}

func NewClientFromAPI(api NotifierAPI) *Client {
	if api == nil {
		return nil
	}
	return &Client{api: api}
}

func NewClientInProcess(api NotifierAPI) *Client {
	return NewClientFromAPI(api)
}

func (c *Client) API() NotifierAPI {
	if c == nil {
		return nil
	}
	return c.api
}

func (c *Client) Close() error {
	if c == nil || c.closeFn == nil {
		return nil
	}
	c.closeFn()
	c.closeFn = nil
	return nil
}
