import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

import clientAppDevicesReducer from '@shared-ui/app/clientApp/Devices/slice'

import devicesReducer from '@/containers/Devices/slice'
import notificationsReducer from '@/containers/Notifications/slice'
import appReducer from '@/containers/App/slice'
import remoteClientsReducer from '@/containers/RemoteClients/slice'

/**
 * @description
 * Combines all reducers required by the app.
 */
export const createRootReducer = () =>
    enableBatching(
        combineReducers({
            app: appReducer,
            remoteClients: remoteClientsReducer,
            devices: devicesReducer,
            clientAppDevices: clientAppDevicesReducer,
            notifications: notificationsReducer,
        })
    )
