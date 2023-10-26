package pb

func (r *GetPendingCommandsRequest) ConvertHTTPResourceIDFilter() []*ResourceIdFilter {
	return ResourceIdFilterFromString(r.GetHttpResourceIdFilter())
}
