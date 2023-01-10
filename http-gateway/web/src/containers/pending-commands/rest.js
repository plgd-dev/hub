import { fetchApi, security } from '@/common/services'

import { pendingCommandsApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'

/**
 * Cancel a pending command Rest Api endpoint
 * @param {*} params { deviceId, href, correlationId }
 * @param {*} data
 */
export const cancelPendingCommandApi = ({
  deviceId,
  href = null,
  correlationId,
}) => {
  const { httpGatewayAddress } = security.getGeneralConfig()

  // If the href is provided, it is a resource pending command.
  // If the href is not provided, it is a metadata update pending command.
  if (href) {
    return fetchApi(
      `${httpGatewayAddress}${pendingCommandsApiEndpoints.PENDING_COMMANDS}?resourceId.deviceId=${deviceId}&resourceId.href=${href}&correlationIdFilter=${correlationId}`,
      { method: 'DELETE' }
    )
  }

  return fetchApi(
    `${httpGatewayAddress}${devicesApiEndpoints.DEVICES}/${deviceId}/pending-metadata-updates?correlationIdFilter=${correlationId}`,
    { method: 'DELETE' }
  )
}
