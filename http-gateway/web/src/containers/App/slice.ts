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
        previewTheme: any
    }
}

const initialState: StoreType = {
    version: {},
    configuration: {
        theme: '',
        themes: [],
        previewTheme: {},
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
        setPreviewTheme(state, { payload }) {
            state.configuration.previewTheme = payload
        },
    },
})

// Actions
export const { setVersion, setThemes, setTheme, setPreviewTheme } = actions

// Reducer
export default reducer
