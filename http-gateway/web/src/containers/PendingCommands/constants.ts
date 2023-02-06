export const pendingCommandsApiEndpoints = {
  PENDING_COMMANDS: '/api/v1/pending-commands',
}

export const PENDING_COMMANDS_DEFAULT_PAGE_SIZE = 10

export const EMBEDDED_PENDING_COMMANDS_DEFAULT_PAGE_SIZE = 5

export const commandTypes = {
  RESOURCE_CREATE_PENDING: 'resourceCreatePending',
  RESOURCE_RETRIEVE_PENDING: 'resourceRetrievePending',
  RESOURCE_UPDATE_PENDING: 'resourceUpdatePending',
  RESOURCE_DELETE_PENDING: 'resourceDeletePending',
  DEVICE_METADATA_UPDATE_PENDING: 'deviceMetadataUpdatePending',
}

export const updatedCommandTypes = {
  RESOURCE_CREATED: 'resourceCreated',
  RESOURCE_DELETED: 'resourceDeleted',
  RESOURCE_UPDATED: 'resourceUpdated',
  RESOURCE_RETRIEVED: 'resourceRetrieved',
  DEVICE_METADATA_UPDATED: 'deviceMetadataUpdated',
}

export const UPDATE_PENDING_COMMANDS_WS_KEY = 'update-pending-commands'
export const NEW_PENDING_COMMAND_WS_KEY = 'new-pending-command'

export const pendingCommandStatuses = {
  UNKNOWN: 'UNKNOWN',
  OK: 'OK',
  BAD_REQUEST: 'BAD_REQUEST',
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  NOT_FOUND: 'NOT_FOUND',
  UNAVAILABLE: 'UNAVAILABLE',
  NOT_IMPLEMENTED: 'NOT_IMPLEMENTED',
  ACCEPTED: 'ACCEPTED',
  ERROR: 'ERROR',
  METHOD_NOT_ALLOWED: 'METHOD_NOT_ALLOWED',
  CREATED: 'CREATED',
  CANCELED: 'CANCELED',
}

export const PENDING_COMMANDS_LIST_REFRESH_INTERVAL_MS = 7000

export const dateFormat = {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
}

export const timeFormat = {
  hour: 'numeric',
  minute: 'numeric',
}

export const timeFormatLong = {
  hour: 'numeric',
  minute: 'numeric',
  second: 'numeric',
}
