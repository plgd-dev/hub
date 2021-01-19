import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

// import devicesReducer from '@/components/devices/slice'

/**
 * @description
 * Combines all reducers required by our app.
 */
export const createRootReducer = () =>
  enableBatching(
    combineReducers({}),
  )
