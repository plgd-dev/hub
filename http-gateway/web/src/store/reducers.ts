import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

import devicesReducer from '@/containers/Devices/slice'
import notificationsReducer from '@/containers/Notifications/slice'
import appReducer from '@/containers/App/slice'

/**
 * @description
 * Combines all reducers required by the app.
 */
export const createRootReducer = () =>
    enableBatching(
        combineReducers({
            app: appReducer,
            devices: devicesReducer,
            notifications: notificationsReducer,
        })
    )
