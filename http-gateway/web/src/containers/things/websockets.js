import { thingsApiEndpoints } from './constants'

import { store, history } from '@/store'
import { Emitter } from '@/common/services/emitter'
import { showInfoToast } from '@/components/toast'
import { thingsStatuses, THINGS_STATUS_WS_KEY } from './constants'
import { selectActiveNotifications } from './slice'
import { getThingApi } from './rest'
import { messages as t } from './things-i18n'

export const deviceStatusListener = async message => {
  const notificationsEnabled = selectActiveNotifications(
    store.getState()
  )?.includes(THINGS_STATUS_WS_KEY)

  setTimeout(async () => {
    const { deviceIds, status } = JSON.parse(message.data)
    try {
      deviceIds.forEach(async deviceId => {
        // Emit an event
        Emitter.emit(`${THINGS_STATUS_WS_KEY}.${deviceId}`, {
          deviceId,
          status,
        })

        // Show toast
        if (notificationsEnabled && status !== thingsStatuses.UNREGISTERED) {
          const {
            data: { device: { n: deviceName } = {} } = {},
          } = await getThingApi(deviceId)
          const toastMessage =
            status === thingsStatuses.ONLINE
              ? t.thingWentOnline
              : t.thingWentOffline
          showInfoToast(
            {
              title: t.thingStatusChange,
              message: { message: toastMessage, params: { name: deviceName } },
            },
            {
              onClick: () => {
                history.push(`/things/${deviceId}`)
              },
            }
          )
        }
      })
    } catch (error) {} // ignore error
  }, notificationsEnabled ? 2000 : 0)
}

export const thingsWSClient = {
  name: THINGS_STATUS_WS_KEY,
  api: thingsApiEndpoints.THINGS_WS,
  listener: deviceStatusListener,
}
