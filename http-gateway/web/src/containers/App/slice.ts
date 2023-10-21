import { createSlice } from '@reduxjs/toolkit'
import { PlgdThemeType } from '@shared-ui/components/Atomic/_theme'

export type StoreType = {
    version: {
        latest?: string
        latest_url?: string
        requestedDatetime?: string
    }
    configuration: {
        theme: PlgdThemeType
    }
}

const initialState: StoreType = {
    version: {},
    configuration: {
        theme: 'light',
    },
}

const { reducer, actions } = createSlice({
    name: 'app',
    initialState,
    reducers: {
        setVersion(state, { payload }) {
            state.version = payload
        },
        setTheme(state, { payload }) {
            state.configuration.theme = payload
        },
    },
})

// Actions
export const { setVersion, setTheme } = actions

// Reducer
export default reducer
