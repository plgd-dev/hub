export const thingsStatuses = {
  ONLINE: 'ONLINE',
  OFFLINE: 'OFFLINE',
  REGISTERED: 'REGISTERED',
  UNREGISTERED: 'UNREGISTERED',
}

export const thingsApiEndpoints = {
  THINGS: '/api/v1/devices',
  THINGS_RESOURCES: '/api/v1/resource-links',
  THINGS_WS: '/api/v1/ws/devices',
}

export const RESOURCES_DEFAULT_PAGE_SIZE = 5

export const THINGS_DEFAULT_PAGE_SIZE = 10

export const RESOURCE_TREE_DEPTH_SIZE = 15 // px

export const errorCodes = {
  DEADLINE_EXCEEDED: 'DeadlineExceeded',
  INVALID_ARGUMENT: 'InvalidArgument',
}

export const resourceModalTypes = {
  UPDATE_RESOURCE: 'update',
  CREATE_RESOURCE: 'create',
}

export const resourceEventTypes = {
  ADDED: 'added',
  REMOVED: 'removed',
}

export const knownInterfaces = {
  OIC_IF_A: 'oic.if.a',
  OIC_IF_BASELINE: 'oic.if.baseline',
  OIC_IF_CREATE: 'oic.if.create',
}

export const knownResourceTypes = {
  OIC_WK_CON: 'oic.wk.con', // contains device name
}

export const defaultNewResource = {
  rt: [],
  if: [knownInterfaces.OIC_IF_A, knownInterfaces.OIC_IF_BASELINE],
  rep: {},
  p: {
    bm: 3,
  },
}

export const NO_DEVICE_NAME = '<no-name>'

// Websocket keys
export const THINGS_WS_KEY = 'things'
export const STATUS_WS_KEY = 'status'
export const RESOURCE_WS_KEY = 'resource'
export const THINGS_STATUS_WS_KEY = `${THINGS_WS_KEY}.${STATUS_WS_KEY}`
export const THINGS_RESOURCE_REGISTRATION_WS_KEY = `${THINGS_WS_KEY}.${RESOURCE_WS_KEY}.registration`
export const THINGS_RESOURCE_UPDATE_WS_KEY = `${THINGS_WS_KEY}.${RESOURCE_WS_KEY}.update`

// Emitter Event keys
export const THINGS_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY =
  'things-registered-unregistered-count'
