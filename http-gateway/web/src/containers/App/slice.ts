import { createSlice } from '@reduxjs/toolkit'

export type StoreType = {
    routeBeforeSingIn?: string
}

const initialState: StoreType = {}

const { reducer, actions } = createSlice({
    name: 'app',
    initialState,
    reducers: {
        setRouterBeforeSignIn: (state, { payload }) => ({ ...state, routeBeforeSingIn: payload }),
    },
})

// Actions
export const { setRouterBeforeSignIn } = actions

// Reducer
export default reducer
