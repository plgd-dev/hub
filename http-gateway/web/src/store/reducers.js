import { combineReducers } from 'redux'
import { enableBatching } from 'redux-batched-actions'

import thingsReducer from '@/containers/things/slice'

/**
 * @description
 * Combines all reducers required by the app.
 */
export const createRootReducer = () =>
  enableBatching(
    combineReducers({
      things: thingsReducer,
    })
  )
