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
        themeModal: boolean
        previewTheme: any
    }
}

const initialState: StoreType = {
    version: {},
    configuration: {
        theme: '',
        themes: [],
        themeModal: false,
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
            if (Object.hasOwn(state, 'configuration')) {
                state.configuration.themes = payload
            } else {
                state.configuration = { ...initialState.configuration, themes: payload }
            }
        },
        setTheme(state, { payload }) {
            state.configuration.theme = payload
        },
        setPreviewTheme(state, { payload }) {
            state.configuration.previewTheme = payload
        },
        setThemeModal(state, { payload }) {
            state.configuration.themeModal = payload
        },
    },
})

// Actions
export const { setVersion, setThemes, setTheme, setPreviewTheme, setThemeModal } = actions

// Reducer
export default reducer
