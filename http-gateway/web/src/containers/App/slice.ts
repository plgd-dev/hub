import { createSlice } from '@reduxjs/toolkit'

export type StoreType = {
    version: {
        latest?: string
        latest_url?: string
        requestedDatetime?: string
    }
}

const initialState: StoreType = {
    version: {},
}

const { reducer, actions } = createSlice({
    name: 'app',
    initialState,
    reducers: {
        setVersion(state, { payload }) {
            state.version = payload
        },
    },
})

// Actions
export const { setVersion } = actions

// Reducer
export default reducer
