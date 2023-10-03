package pb

func (r *GetEventsRequest) ConvertHTTPResourceIDFilter() []*ResourceIdFilter {
	return ResourceIdFilterFromString(r.GetHttpResourceIdFilter())
}
