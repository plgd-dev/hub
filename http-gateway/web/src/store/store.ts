import { configureStore } from '@reduxjs/toolkit'
import { setupListeners } from '@reduxjs/toolkit/query'
import { persistReducer } from 'redux-persist'
import storage from 'redux-persist/lib/storage'

import { createRootReducer } from './reducers'
import { StoreType as NotificationStoreType } from '../containers/Notifications/slice'
import { StoreType as DeviceStoreType } from '../containers/Devices/slice'

export type CombinatedStoreType = {
    notifications: NotificationStoreType
    activeNotifications: DeviceStoreType
}

const persistConfig = {
    key: 'root',
    storage: storage,
    blacklist: [],
}

const rootReducer = createRootReducer()

const persistedReducer = persistReducer(persistConfig, rootReducer)

const store = configureStore({
    reducer: persistedReducer,
    middleware: (getDefaultMiddleware) =>
        getDefaultMiddleware({
            serializableCheck: false,
        }),
})

setupListeners(store.dispatch)

export default store
