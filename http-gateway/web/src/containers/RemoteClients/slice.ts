import { createSlice } from '@reduxjs/toolkit'
import { remoteClientStatuses } from '@/containers/RemoteClients/contacts'
import get from 'lodash/get'

export type RemoteClientStatusType = (typeof remoteClientStatuses)[keyof typeof remoteClientStatuses]

export type RemoteClientType = {
    id: string
    clientName: string
    clientUrl: string
    created: string
    status: RemoteClientStatusType
    version: string
}

export type StoreType = {
    remoteClients: RemoteClientType[]
}

const isDev = get(process.env, 'REACT_APP_TEST_REMOTE_CLIENTS_MOCK_DATA', false)

const initialState: StoreType = {
    remoteClients: isDev
        ? [
              {
                  id: 'FAj9X1Gxs9rLm-62r6yGJ',
                  created: '2023-07-22T17:58:11.427Z',
                  version: '0.6.0',
                  clientName: 'Test',
                  clientUrl: 'https://212.89.237.161:50080',
                  status: remoteClientStatuses.REACHABLE,
              },
              {
                  id: 'ascaa-62r6yGJ',
                  created: '2023-07-22T17:58:11.427Z',
                  version: '0.6.0',
                  clientName: 'Test no working',
                  clientUrl: 'https://localhost:50080',
                  status: remoteClientStatuses.REACHABLE,
              },
              {
                  id: 'ascaa-62r6yGJpop',
                  created: '2023-07-22T17:58:11.427Z',
                  version: '0.6.0',
                  clientName: 'PM local',
                  clientUrl: 'http://localhost:3001',
                  status: remoteClientStatuses.REACHABLE,
              },
          ]
        : [],
}

const { reducer, actions } = createSlice({
    name: 'remoteClients',
    initialState,
    reducers: {
        addRemoteClient(state, { payload }) {
            state.remoteClients.push(payload)
        },
        deleteRemoteClients(state, { payload }) {
            payload.forEach((remoteClientId: string) => {
                state.remoteClients.splice(
                    state.remoteClients.findIndex((remoteClient) => remoteClient.id === remoteClientId),
                    1
                )
            })
        },
        updateRemoteClients(state, { payload }) {
            payload.forEach((remoteClient: RemoteClientType) => {
                const index = state.remoteClients.findIndex((originRemoteClient) => originRemoteClient.id === remoteClient.id)

                if (index >= 0) {
                    state.remoteClients[index] = remoteClient
                }
            })
        },
    },
})

// Actions
export const { addRemoteClient, deleteRemoteClients, updateRemoteClients } = actions

// Reducer
export default reducer
