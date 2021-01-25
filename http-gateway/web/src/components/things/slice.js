// This state holds information about devices.

import { createSlice } from '@reduxjs/toolkit'

const initialState = {
  loading: false,
}

const { reducer, actions } = createSlice({
  name: 'things',
  initialState,
  reducers: {
    setLoading(state, { payload }) {
      state.loading = payload
    },
  },
})

// Actions
export const { setLoading } = actions

// Reducer
export default reducer

// Selectors
export const selectThingsLoading = state => state.loading
