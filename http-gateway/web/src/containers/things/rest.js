import { fetchApi, security } from '@/common/services'

import { thingsApiEndpoints } from './constants'
import { interfaceGetParam } from './utils'

/**
 * Get a things Rest Api endpoint
 * @param {*} params { deviceId }
 * @param {*} data
 */
export const getThingsApi = () =>
  fetchApi(`${security.getHttpGatewayAddress()}${thingsApiEndpoints.THINGS}}`)

/**
 * Get a single thing by its ID Rest Api endpoint
 * @param {*} params { deviceId }
 * @param {*} data
 */
export const getThingApi = deviceId =>
  fetchApi(
    `${security.getHttpGatewayAddress()}${
      thingsApiEndpoints.THINGS
    }/${deviceId}`
  )

/**
 * Get things RESOURCES Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface }
 * @param {*} data
 */
export const getThingsResourcesApi = ({
  deviceId,
  href,
  currentInterface = null,
}) =>
  fetchApi(
    `${security.getHttpGatewayAddress()}${
      thingsApiEndpoints.THINGS
    }/${deviceId}/resources${href}${interfaceGetParam(currentInterface)}`
  )

/**
 * Update things RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface }
 * @param {*} data
 */
export const updateThingsResourceApi = (
  { deviceId, href, currentInterface = null },
  data
) => {
  return fetchApi(
    `${security.getHttpGatewayAddress()}${
      thingsApiEndpoints.THINGS
    }/${deviceId}/resources${href}${interfaceGetParam(currentInterface)}`,
    { method: 'PUT', body: data }
  )
}

/**
 * Create things RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface }
 * @param {*} data
 */
export const createThingsResourceApi = (
  { deviceId, href, currentInterface = null },
  data
) => {
  return fetchApi(
    `${security.getHttpGatewayAddress()}${
      thingsApiEndpoints.THINGS
    }/${deviceId}/resource-links${href}${interfaceGetParam(currentInterface)}`,
    { method: 'POST', body: data }
  )
}

/**
 * Delete things RESOURCE Rest Api endpoint
 * @param {*} params { deviceId, href - resource href, currentInterface - interface }
 * @param {*} data
 */
export const deleteThingsResourceApi = ({ deviceId, href }) => {
  return fetchApi(
    `${security.getHttpGatewayAddress()}${
      thingsApiEndpoints.THINGS
    }/${deviceId}/resource-links${href}`,
    { method: 'DELETE' }
  )
}
