import { configureStore } from '@reduxjs/toolkit'
import { setupListeners } from '@reduxjs/toolkit/query'
import { PERSIST, persistReducer, PURGE, REHYDRATE } from 'redux-persist'
import storage from 'redux-persist/lib/storage'
import { createStateSyncMiddleware, initMessageListener } from 'redux-state-sync'

import { createRootReducer } from './reducers'
import { StoreType as NotificationStoreType } from '../containers/Notifications/slice'
import { StoreType as DeviceStoreType } from '../containers/Devices/slice'
import { StoreType as AppStoreType } from '../containers/App/slice'
import { StoreType as RemoteClientType } from '../containers/RemoteClients/slice'

export type CombinedStoreType = {
    notifications: NotificationStoreType
    activeNotifications: DeviceStoreType
    app: AppStoreType
    remoteClients: RemoteClientType
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
            immutableCheck: false,
        }).concat(
            createStateSyncMiddleware({
                predicate: (action) => {
                    // console.log({ action })
                    const blacklist = [PERSIST, PURGE, REHYDRATE]
                    if (typeof action !== 'function') {
                        if (Array.isArray(blacklist)) {
                            return blacklist.indexOf(action.type) < 0
                        }
                    }
                    return false
                },
            })
        ),
})

initMessageListener(store)
setupListeners(store.dispatch)

export default store
