import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

import devicesReducer from '@/containers/Devices/slice'

/**
 * @description
 * Combines all reducers required by the app.
 */
export const createRootReducer = () =>
    enableBatching(
        combineReducers({
            devices: devicesReducer,
        })
    )
