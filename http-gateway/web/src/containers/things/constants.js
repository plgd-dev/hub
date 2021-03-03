export const thingsStatuses = {
  ONLINE: 'online',
  OFFLINE: 'offline',
  REGISTERED: 'registered',
  UNREGISTERED: 'unregistered',
}

export const thingsApiEndpoints = {
  THINGS: '/api/v1/devices',
  THINGS_WS: '/api/v1/ws/devices',
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

export const THINGS_WS_KEY = 'things'
export const STATUS_WS_KEY = 'status'
export const RESOURCE_WS_KEY = 'resource'

export const THINGS_STATUS_WS_KEY = `${THINGS_WS_KEY}.${STATUS_WS_KEY}`
