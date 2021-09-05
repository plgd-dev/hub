import { fetchApi, security } from '@/common/services'

import { pendingCommandsApiEndpoints } from './constants'

/**
 * Cancel a pending command Rest Api endpoint
 * @param {*} params { deviceId, href, correlationId }
 * @param {*} data
 */
export const cancelPendingCommandApi = ({ deviceId, href, correlationId }) =>
  fetchApi(
    `${security.getHttpGatewayAddress()}${
      pendingCommandsApiEndpoints.PENDING_COMMANDS
    }?resourceId.deviceId=${deviceId}&resourceId.href=${href}&correlationIdFilter=${correlationId}`,
    { method: 'DELETE' }
  )
