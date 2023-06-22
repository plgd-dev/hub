package pb

import commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"

func (e *Event_ResourceChanged) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceChanged.GetResourceId()
}

func (e *Event_ResourceUpdatePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceUpdatePending.GetResourceId()
}

func (e *Event_ResourceUpdated) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceUpdated.GetResourceId()
}

func (e *Event_ResourceRetrievePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceRetrievePending.GetResourceId()
}

func (e *Event_ResourceRetrieved) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceRetrieved.GetResourceId()
}

func (e *Event_ResourceDeletePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceDeletePending.GetResourceId()
}

func (e *Event_ResourceDeleted) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceDeleted.GetResourceId()
}

func (e *Event_ResourceCreatePending) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceCreatePending.GetResourceId()
}

func (e *Event_ResourceCreated) GetResourceId() *commands.ResourceId {
	if e == nil {
		return nil
	}
	return e.ResourceCreated.GetResourceId()
}
