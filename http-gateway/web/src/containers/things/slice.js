// This state holds information about devices.

import { createSlice } from '@reduxjs/toolkit'

const initialState = {
  activeNotifications: [],
}

const { reducer, actions } = createSlice({
  name: 'things',
  initialState,
  reducers: {
    addActiveNotification(state, { payload }) {
      state.activeNotifications.push(payload)
    },
    deleteActiveNotification(state, { payload }) {
      state.activeNotifications.splice(
        state.activeNotifications.findIndex(
          notification => notification === payload
        ),
        1
      )
    },
    toggleActiveNotification(state, { payload }) {
      if (!state.activeNotifications.includes(payload)) {
        state.activeNotifications.push(payload)
      } else {
        state.activeNotifications.splice(
          state.activeNotifications.findIndex(
            notification => notification === payload
          ),
          1
        )
      }
    },
  },
})

// Actions
export const {
  addActiveNotification,
  deleteActiveNotification,
  toggleActiveNotification,
} = actions

// Reducer
export default reducer

// Selectors
export const selectActiveNotifications = state =>
  state.things.activeNotifications
