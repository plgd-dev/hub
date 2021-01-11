import { configureStore } from '@reduxjs/toolkit'

import { createRootReducer } from './reducers'

const store = configureStore({
  reducer: createRootReducer(),
})

export default store
