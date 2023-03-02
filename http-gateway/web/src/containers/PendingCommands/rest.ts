import { fetchApi, security } from '@shared-ui/common/services'

import { pendingCommandsApiEndpoints } from './constants'
import { devicesApiEndpoints } from '@/containers/Devices/constants'
import { SecurityConfig } from '@/containers/App/App.types'

/**
 * Cancel a pending command Rest Api endpoint
 * @param {*} params { deviceId, href, correlationId }
 * @param {*} data
 */
export const cancelPendingCommandApi = ({
  deviceId,
  href = undefined,
  correlationId,
}: {
  deviceId: string
  href?: string
  correlationId?: string
}) => {
  const { httpGatewayAddress, cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
  // If the href is provided, it is a resource pending command.
  // If the href is not provided, it is a metadata update pending command.
  if (href) {
    return fetchApi(
      `${httpGatewayAddress}${
        pendingCommandsApiEndpoints.PENDING_COMMANDS
      }?resourceId.deviceId=${deviceId}&resourceId.href=${href}&correlationIdFilter=${correlationId}`,
      { method: 'DELETE', cancelRequestDeadlineTimeout }
    )
  }

  return fetchApi(
    `${httpGatewayAddress}${
      devicesApiEndpoints.DEVICES
    }/${deviceId}/pending-metadata-updates?correlationIdFilter=${correlationId}`,
    { method: 'DELETE', cancelRequestDeadlineTimeout }
  )
}
