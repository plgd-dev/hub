package pb

func (cr *Configuration_Resource) Clone() *Configuration_Resource {
	if cr == nil {
		return nil
	}
	return &Configuration_Resource{
		Href:       cr.GetHref(),
		Content:    cr.GetContent().Clone(),
		TimeToLive: cr.GetTimeToLive(),
	}
}

func (c *Configuration) Clone() *Configuration {
	cfg := &Configuration{
		Id:        c.GetId(),
		Version:   c.GetVersion(),
		Name:      c.GetName(),
		Owner:     c.GetOwner(),
		Timestamp: c.GetTimestamp(),
	}
	for _, r := range c.GetResources() {
		cfg.Resources = append(cfg.Resources, r.Clone())
	}
	return cfg
}
