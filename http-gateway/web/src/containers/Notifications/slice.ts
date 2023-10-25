import { createSlice } from '@reduxjs/toolkit'
import { NotificationCenterItemType } from '@shared-ui/components/Atomic/NotificationCenter/NotificationCenter.types'

export type StoreType = []

const initialState: NotificationCenterItemType[] = []

const maxNotificationCount = 25

const { reducer, actions } = createSlice({
    name: 'notifications',
    initialState,
    reducers: {
        readAllNotifications: (state) => state?.map((notification: any) => ({ ...notification, read: true })),
        setNotifications: (state, { payload }) => payload.slice(0, maxNotificationCount).map((n: any) => ({ ...n, content: undefined })),
        cleanNotifications: () => initialState,
    },
})

// Actions
export const { readAllNotifications, setNotifications, cleanNotifications } = actions

// Reducer
export default reducer
