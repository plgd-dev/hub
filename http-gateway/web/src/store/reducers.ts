import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

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
            notifications: notificationsReducer,
        })
    )
