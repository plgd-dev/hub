export const thingsStatuses = {
  ONLINE: 'online',
  OFFLINE: 'offline',
}

export const thingsApiEndpoints = {
  THINGS: '/api/v1/devices',
}

export const RESOURCES_DEFAULT_PAGE_SIZE = 5

export const THINGS_DEFAULT_PAGE_SIZE = 10

export const errorCodes = {
  DEADLINE_EXCEEDED: 'DeadlineExceeded',
  INVALID_ARGUMENT: 'InvalidArgument',
}

export const resourceModalTypes = {
  UPDATE_RESOURCE: 'update',
  CREATE_RESOURCE: 'create',
}

export const knownInterfaces = {
  OIC_IF_A: 'oic.if.a',
  OIC_IF_BASELINE: 'oic.if.baseline',
  OIC_IF_CREATE: 'oic.if.create',
}

export const defaultNewResource = {
  rt: [],
  if: [knownInterfaces.OIC_IF_A, knownInterfaces.OIC_IF_BASELINE],
  rep: {},
  p: {
    bm: 3,
  },
}
