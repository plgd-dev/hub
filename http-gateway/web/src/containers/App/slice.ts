import { createSlice } from '@reduxjs/toolkit'

export type StoreType = {
    version: {
        latest?: string
        latest_url?: string
        requestedDatetime?: string
    }
    configuration: {
        theme: string
        themes: string[]
    }
}

const initialState: StoreType = {
    version: {},
    configuration: {
        theme: '',
        themes: [],
    },
}

const { reducer, actions } = createSlice({
    name: 'app',
    initialState,
    reducers: {
        setVersion(state, { payload }) {
            state.version = payload
        },
        setThemes(state, { payload }) {
            state.configuration.themes = payload
        },
        setTheme(state, { payload }) {
            state.configuration.theme = payload
        },
    },
})

// Actions
export const { setVersion, setThemes, setTheme } = actions

// Reducer
export default reducer
